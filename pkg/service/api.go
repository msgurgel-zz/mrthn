/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/gorilla/context"

	"github.com/msgurgel/marathon/pkg/auth"
	"github.com/msgurgel/marathon/pkg/environment"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"

	"github.com/msgurgel/marathon/pkg/model"
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
	clientId, err := strconv.Atoi(idStr)

	// Generate random secret
	secret := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		api.log.WithFields(logrus.Fields{
			"id":  clientId,
			"err": err,
		}).Error("failed to generate secret token")

		respondWithError(w, http.StatusInternalServerError, "Something went wrong. Try again later...")
		return
	}

	// Store secret in the DB as part of the Client table
	rows, err := dal.InsertSecretInExistingClient(api.db, clientId, secret)
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
			"clientId": clientId,
		}).Warn("received /get-token request with invalid client ID")

		respondWithError(w, http.StatusBadRequest, "client ID does not exist")
		return
	}

	// Add client ID as part of the JWT claims
	tokenString, _ := generateJWT(clientId, secret)

	// Send the token back to the requestor
	_, err = w.Write([]byte(tokenString))
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to send JWT")
	}
}

func (api *Api) GetUserCalories(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, _ := strconv.Atoi(vars["userID"]) // TODO: deal with error

	response := GetUserCaloriesResponse200{
		Id:       userId,
		Calories: model.GetUserCalories(userId),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) GetUserSteps(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId, _ := strconv.Atoi(vars["userID"]) // TODO: deal with error

	response := GetUserStepsResponse200{
		Id:    userId,
		Steps: model.GetUserSteps(userId),
	}
	respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) Login(w http.ResponseWriter, r *http.Request) {

	// check what login we actually want to authorize into
	service, serviceOk := r.URL.Query()["service"]
	callBackURL, callbackOk := r.URL.Query()["callback"]

	if !serviceOk || len(service) != 1 {
		// they didn't put the service code in properly
		respondWithError(w, http.StatusBadRequest, "expected single 'service' parameter with name of service to authenticate with")
	} else if !callbackOk || len(callBackURL) != 1 {
		respondWithError(w, http.StatusBadRequest, "expected single 'callback' parameter to contain valid callback url")
	} else {

		// get the client id
		clientId := context.Get(r, "client_id").(int)

		// create the state object
		RequestStateObject, ok := api.authMethods.Oauth2.CreateStateObject(callBackURL[0], service[0], clientId)

		if ok == nil {
			// check what type of request was made using the StateObject

			// redirect with the stateObjects url
			url := RequestStateObject.URL
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		}
	}

}

func (api *Api) Callback(w http.ResponseWriter, r *http.Request) {
	// Check that the state returned was valid
	Oauth2Result, err := api.authMethods.Oauth2.ObtainUserTokens(r.FormValue("state"), r.FormValue("code"))

	if err == nil {

		userId, err := createUser(&Oauth2Result, api.db, api.log)

		if err != nil {

			var jsonStr = []byte(`{"error":"` + err.Error() + `"}`)
			api.sendAuthorizationResult(jsonStr, Oauth2Result.Callback)
		} else {
			jsonStr := []byte(`{"userId":"` + string(userId) + `"}`)
			api.sendAuthorizationResult(jsonStr, Oauth2Result.Callback)
		}

	} else {
		// Something went wrong, instead of the result, send back the error
		var jsonStr = []byte(`{"error":"` + err.Error() + `"}`)
		api.sendAuthorizationResult(jsonStr, Oauth2Result.Callback)
	}

}

func (api *Api) sendAuthorizationResult(body []byte, Callback string) {
	req, _ := http.NewRequest("POST", Callback, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	api.log.Info("sending authorization result [" + string(body) + "] to [" + Callback + "]")
	callbackResponse, err := http.Post(Callback, "application/json", bytes.NewBuffer(body))

	if err != nil {
		// log the error
		api.log.Error(err)
		return
	}

	defer callbackResponse.Body.Close()
}

// Helpers Functions

// TODO: move this somewhere else?
func createUser(OauthParams *auth.OAuthResult, db *sql.DB, log *logrus.Logger) (int, error) {
	// check what kind of service this user is being created for
	switch OauthParams.Service {
	case "fitbit":

		// before we create the user, check the id to see if its in the database

		userId, err := CheckFitBitUser(db, OauthParams)

		if err != nil {
			return 0, err
		}

		if userId != 0 {
			// this user already exists, just return the userId
			return userId, nil
		}

		// make the fitbit user and return the userId
		userId, err = dal.CreateFitbitUser(db, OauthParams)

		if err != nil {

			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("failed to create a new fitbit user")
			return 0, err
		} else {
			log.WithFields(logrus.Fields{
				"userId": userId,
			}).Info("new fitbit user created")
			return userId, nil
		}
	default:
		return 0, errors.New(OauthParams.Service + " service does not exist")
	}

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
