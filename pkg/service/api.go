/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/msgurgel/marathon/pkg/auth"
	"github.com/msgurgel/marathon/pkg/dal"
	"github.com/msgurgel/marathon/pkg/environment"
	"github.com/msgurgel/marathon/pkg/helpers"
	"github.com/msgurgel/marathon/pkg/model"

	"github.com/sirupsen/logrus"
)

type Api struct {
	log         *logrus.Logger
	authMethods auth.Types
	db          *sql.DB
}

func NewApi(db *sql.DB, logger *logrus.Logger, config *environment.MarathonConfig) Api {
	api := Api{
		log: logger,
		db:  db,
	}
	api.authMethods.Init(config)

	return api
}

func (api *Api) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func (api *Api) GetToken(w http.ResponseWriter, r *http.Request) {
	// Get Client ID from request (check if clientID is in db)
	idStr := r.FormValue("id")
	clientID, err := strconv.Atoi(idStr)

	// Generate random secret
	secret := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		api.log.WithFields(logrus.Fields{
			"id":  clientID,
			"err": err,
		}).Error("failed to generate secret token")

		respondWithError(w, http.StatusInternalServerError, "Something went wrong. Try again later...")
		return
	}

	// Store secret in the DB as part of the Client table
	rows, err := dal.InsertSecretInExistingClient(api.db, clientID, secret)
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to update client with new generated secret")

		respondWithError(w, http.StatusInternalServerError, "Something went wrong. Try again later...")
		return
	}

	// Make sure that we updated the client with the new secret
	if rows != 1 {
		api.log.WithFields(logrus.Fields{
			"clientID": clientID,
		}).Warn("received /get-token request with invalid client ID")

		respondWithError(w, http.StatusBadRequest, "client ID does not exist")
		return
	}

	// Add client ID as part of the JWT claims
	tokenString, _ := generateJWT(clientID, secret)

	// Send the token back to the requestor
	_, err = w.Write([]byte(tokenString))
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to send JWT")
	}
}

func (api *Api) GetUserCalories(w http.ResponseWriter, r *http.Request) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func": "GetUserCalories"})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	caloriesValues, err := model.GetUserCalories(api.db, api.log, userID, date)
	if err != nil {
		// TODO: Change this to a more fitting HTTP code
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetUserCaloriesResponse200{
		ID:       userID,
		Calories: caloriesValues,
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) GetUserSteps(w http.ResponseWriter, r *http.Request) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func:": "GetUserSteps"})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	stepsValues, err := model.GetUserSteps(api.db, api.log, userID, date)
	if err != nil {
		// TODO: Change this to a more fitting HTTP code
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetUserStepsResponse200{
		ID:    userID,
		Steps: stepsValues,
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) Login(w http.ResponseWriter, r *http.Request) {
	service, serviceOk := r.URL.Query()["service"]
	callBackURL, callbackOk := r.URL.Query()["callback"]

	if !serviceOk || len(service) != 1 {
		api.log.WithFields(logrus.Fields{
			"func": "Login",
		}).Error("missing URL param 'service'")

		respondWithError(w, http.StatusBadRequest, "expected single 'service' parameter with name of service to authenticate with")
		return
	}

	if !callbackOk || len(callBackURL) != 1 {
		api.log.WithFields(logrus.Fields{
			"func": "Login",
		}).Error("missing URL param 'callback'")

		respondWithError(w, http.StatusBadRequest, "expected single 'callback' parameter to contain valid callback url")
		return
	}

	// Get client ID from the context, set during the authentication phase
	clientID := context.Get(r, "client_id").(int)

	// Create the state object TODO: This is dependent on OAuth2. When new auth types are needed, this will have to be changed
	RequestStateObject, ok := api.authMethods.Oauth2.CreateStateObject(callBackURL[0], service[0], clientID)

	if ok == nil {
		url := RequestStateObject.URL                          // check what type of request was made using the StateObject
		http.Redirect(w, r, url, http.StatusTemporaryRedirect) // redirect with the stateObjects url
	}
}

func (api *Api) Callback(w http.ResponseWriter, r *http.Request) {
	// Check that the state returned was valid TODO: Remove dependency on OAuth2
	Oauth2Result, err := api.authMethods.Oauth2.ObtainUserTokens(r.FormValue("state"), r.FormValue("code"))
	if err != nil {
		// Something went wrong, instead of the result, send back the error
		api.log.WithFields(logrus.Fields{
			"func":  "Callback",
			"err":   err,
			"state": r.FormValue("state"),
		}).Error("failed to retrieve Oauth2 token for user")
		api.sendAuthorizationResult(w, r, 0, Oauth2Result.Callback) // TODO: This goes against Go's design principles. Need to be changed
		return
	}

	userID, err := createUser(&Oauth2Result, api.db, api.log)
	if err != nil {

		api.log.WithFields(logrus.Fields{
			"func": "Callback",
			"err":  err,
		}).Error("failed to create a new user in the database")

		api.sendAuthorizationResult(w, r, userID, Oauth2Result.Callback)
	} else {
		api.sendAuthorizationResult(w, r, userID, Oauth2Result.Callback)
	}
}

// Helpers Functions

func (api *Api) sendAuthorizationResult(w http.ResponseWriter, r *http.Request, userId int, Callback string) {

	// add the url parameters to the callback url
	Callback += fmt.Sprintf("?userId=%d", userId)

	api.log.WithFields(logrus.Fields{
		"callback": Callback,
		"userId":   userId,
	}).Info("sending login result to client")

	http.Redirect(w, r, Callback, 307)

}

func (api *Api) getRequestParams(r *http.Request, fields logrus.Fields) (userID int, date time.Time, err error) {
	// Get user from URL
	vars := mux.Vars(r)
	userID, _ = strconv.Atoi(vars["userID"]) // TODO: deal with error

	// Get date from query params
	dateStr, ok := r.URL.Query()["date"]
	if !ok || len(dateStr) != 1 {
		api.log.WithFields(fields).Error("missing URL param 'date'")

		return 0, time.Time{}, errors.New("expected single 'date' parameter")
	}

	date, err = helpers.ParseISODate(dateStr[0])
	if err != nil {
		fields["dateStr"] = dateStr
		fields["err"] = err
		api.log.WithFields(fields).Error("failed to parse date")

		return 0, time.Time{}, errors.New("invalid date")
	}

	return userID, date, err
}

func createUser(Oauth2Params *auth.OAuth2Result, db *sql.DB, log *logrus.Logger) (int, error) {
	// check what kind of service this user is being created for
	switch Oauth2Params.PlatformName {
	case "fitbit":
		// before we create the user, check the id to see if its in the database
		userID, err := dal.GetUserByPlatformID(db, Oauth2Params.PlatformID, Oauth2Params.PlatformName)

		if err != nil {
			return 0, err
		}

		if userID != 0 {
			// this user already exists, just return the userID
			return userID, nil
		}

		// create the credentials for the user
		var connectionParams = []string{"oauth2", Oauth2Params.AccessToken, Oauth2Params.RefreshToken}
		connStr, err := formatConnectionString(connectionParams)
		if err != nil {
			return 0, err
		}

		params := dal.CredentialParams{
			ClientID:         Oauth2Params.ClientID,
			PlatformName:     Oauth2Params.PlatformName,
			UPID:             Oauth2Params.PlatformID,
			ConnectionString: connStr,
		}
		userID, err = dal.InsertUserCredentials(db, params)

		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("failed to create a new fitbit user")
			return 0, err
		} else {
			log.WithFields(logrus.Fields{
				"userID": userID,
			}).Info("new fitbit user created")
			return userID, nil
		}
	default:
		return 0, errors.New(Oauth2Params.PlatformName + " service does not exist")
	}
}

func formatConnectionString(connectionParams []string) (string, error) {
	if len(connectionParams) == 0 {
		return "", errors.New("must contain non zero amount of Connection parameters")
	}

	var sb strings.Builder
	for _, param := range connectionParams {
		sb.WriteString(param + ";")
	}

	return sb.String(), nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response) // TODO: deal with possible error
}
