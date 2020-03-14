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

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/msgurgel/marathon/pkg/auth"
	"github.com/msgurgel/marathon/pkg/dal"
	"github.com/msgurgel/marathon/pkg/environment"
	"github.com/msgurgel/marathon/pkg/helpers"
	"github.com/msgurgel/marathon/pkg/model"
)

const FormMaxMemoryLimit = 128

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
	api.authMethods.GetAuthTypes(config)

	return api
}

func (api *Api) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "API is working ✌️")
}

func (api *Api) GetToken(w http.ResponseWriter, r *http.Request) {
	// Get Client ID from request (check if clientID is in db)
	//idStr := r.FormValue("id")

	idStr, ok := r.URL.Query()["id"]

	if !ok || len(idStr[0]) < 1 {
		api.log.WithFields(logrus.Fields{
			"err": "url parameter 'id' not found in request",
		}).Error("id of client missing/malformed")

		api.respondWithError(w, http.StatusBadRequest,
			"Error: clientId missing")
		return
	}

	clientID, err := strconv.Atoi(idStr[0])

	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("id of client malformed")

		api.respondWithError(w, http.StatusBadRequest,
			"Error: clientId must be an integer")
		return
	}

	// Generate random secret
	secret := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		api.log.WithFields(logrus.Fields{
			"id":  clientID,
			"err": err,
		}).Error("failed to generate secret token")

		api.respondWithError(w, http.StatusInternalServerError,
			"Something went wrong. Try again later...")
		return
	}

	// Store secret in the DB as part of the Client table
	rows, err := dal.InsertSecretInExistingClient(api.db, clientID, secret)
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "GetToken",
			"err":  err,
		}).Error("failed to update client with new generated secret")

		api.respondWithError(w, http.StatusInternalServerError,
			"Something went wrong. Try again later...")
		return
	}

	// Make sure that we updated the client with the new secret
	if rows != 1 {
		api.log.WithFields(logrus.Fields{
			"func":     "GetToken",
			"clientID": clientID,
		}).Warn("received /get-token request with invalid client ID")

		api.respondWithError(w, http.StatusBadRequest, "client ID does not exist")
		return
	}

	// Add client ID as part of the JWT claims
	tokenString, _ := generateJWT(clientID, secret)

	// Send the token back to the requester
	_, err = w.Write([]byte(tokenString))
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"func": "GetToken",
			"err":  err,
		}).Error("failed to send JWT")
	}
}

func (api *Api) GetUserCalories(w http.ResponseWriter, r *http.Request) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func": "GetUserCalories"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
	}

	caloriesValues, err := model.GetUserCalories(api.db, api.log, userID, date)
	if err != nil {
		// TODO: Change this to a more fitting HTTP code
		api.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetUserCaloriesResponse{
		ID:       userID,
		Calories: caloriesValues,
	}
	api.respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) GetUserDistance(w http.ResponseWriter, r *http.Request) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func": "GetUserDistance"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
	}

	distanceValues, err := model.GetUserDistance(api.db, api.log, userID, date)
	if err != nil {
		// TODO: Change this to a more fitting HTTP code
		api.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetUserDistanceResponse{
		ID:       userID,
		Distance: distanceValues,
	}
	api.respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) GetUserSteps(w http.ResponseWriter, r *http.Request) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func:": "GetUserSteps"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	stepsValues, err := model.GetUserSteps(api.db, api.log, userID, date)
	if err != nil {
		// TODO: Change this to a more fitting HTTP code
		api.respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := GetUserStepsResponse{
		ID:    userID,
		Steps: stepsValues,
	}
	api.respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) Login(w http.ResponseWriter, r *http.Request) {
	// Check if the client has passed in a JWT token in the url
	token, tokenOk := r.URL.Query()["token"]
	if !tokenOk || len(token) != 1 {
		api.log.WithFields(logrus.Fields{
			"func": "Login",
		}).Error("missing URL param 'token'")

		api.respondWithError(w, http.StatusBadRequest,
			"Invalid token")
		return
	}

	// If the token exists, verify it exists in the database
	parseToken, err := validateJWT(api.db, token[0])
	if err != nil || !parseToken.valid {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("JWT was invalid")

		api.respondWithError(w, http.StatusUnauthorized, "Invalid JWT token")
		return
	}

	service, serviceOk := r.URL.Query()["service"]
	callBackURL, callbackOk := r.URL.Query()["callback"]

	if !serviceOk || len(service) != 1 {
		api.log.WithFields(logrus.Fields{
			"func": "Login",
		}).Error("missing URL param 'service'")

		api.respondWithError(w, http.StatusBadRequest,
			"expected single 'service' parameter with name of service to authenticate with",
		)
		return
	}

	if !callbackOk || len(callBackURL) != 1 {
		api.log.WithFields(logrus.Fields{
			"func": "Login",
		}).Error("missing URL param 'callback'")

		api.respondWithError(w, http.StatusBadRequest,
			"expected single 'callback' parameter to contain valid callback url")
		return
	}

	// Create the state object TODO: This is dependent on OAuth2. When new auth types are needed, this will have to be changed
	RequestStateObject, ok := api.authMethods.Oauth2.CreateStateObject(callBackURL[0], service[0], parseToken.clientID)

	if ok == nil {
		url := RequestStateObject.URL                          // check what type of request was made using the StateObject
		http.Redirect(w, r, url, http.StatusTemporaryRedirect) // redirect with the stateObjects url
	}
}

func (api *Api) Callback(w http.ResponseWriter, r *http.Request) {
	// TODO: Remove dependency on OAuth2
	// Check that the state returned was valid
	Oauth2Result, err := api.authMethods.Oauth2.ObtainUserTokens(
		r.FormValue("state"),
		r.FormValue("code"),
	)
	if err != nil {
		// Something went wrong. Instead of the result, send back the error
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
	}

	api.sendAuthorizationResult(w, r, userID, Oauth2Result.Callback)
}

func (api *Api) SignUp(w http.ResponseWriter, r *http.Request) {
	// get the new values of the client
	err := r.ParseMultipartForm(FormMaxMemoryLimit)

	if err != nil {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Error occurred while attempting to parse form values",
		}
		api.respondWithJSON(w, http.StatusInternalServerError, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignUp",
			"err":  err,
		}).Error("error occurred while attempting to parse form values")

		return
	}

	clientName := r.Form.Get("name")

	if clientName == "" {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Expected parameter 'name' in request",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)
		api.log.WithFields(logrus.Fields{
			"func": "SignUp",
			"err":  "client name was missing in forms field",
		}).Error("error occurred while attempting to parse form values")

		return
	}

	// Check if the name already exists. This should probably be done already before SignUp is called
	userId, err := dal.CheckClientName(api.db, clientName)
	if err != nil {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Error occurred while processing request",
		}
		api.respondWithJSON(w, http.StatusInternalServerError, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignUp",
			"err":  err,
		}).Error("failed to check client name")

		return
	}

	if userId != 0 {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Client name already taken",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func":       "SignUp",
			"clientName": clientName,
		}).Warn("signup client name already taken")

		return
	}

	clientPassWord := r.Form.Get("password")

	if clientPassWord == "" {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Expected parameter 'password' in request",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignUp",
			"err":  "client password was missing in forms field",
		}).Error("failed to check client password")

		return
	}

	clientID, err := dal.CreateNewClient(api.db, clientName, clientPassWord)
	if err != nil {
		api.respondWithError(w, http.StatusInternalServerError, "Error occurred while attempting to create client")
		api.log.WithFields(logrus.Fields{
			"func": "SignUp",
			"err":  err,
		}).Error("error occurred while attempting to create client")

		return
	}

	// Send a success message back
	response := ClientSignUpResponse{
		Success:    true,
		ClientID:   clientID,
		ClientName: clientName,
	}
	api.respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) SignIn(w http.ResponseWriter, r *http.Request) {
	// TODO: Make sure that this uses ParseForm instead of ParseMultipartForm
	err := r.ParseMultipartForm(FormMaxMemoryLimit)
	if err != nil {
		response := ClientSignUpResponse{
			Success: false,
			Error:   "Error occurred while attempting to parse form values",
		}
		api.respondWithJSON(w, http.StatusInternalServerError, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignIn",
			"err":  err,
		}).Error("failed to parse request form values")

		return
	}

	clientName := r.Form.Get("name")

	if clientName == "" {
		response := ClientSignInResponse{
			Success:  false,
			ClientID: 0,
			Error:    "Expected parameter 'name' in request",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignIn",
			"err":  "form parameter 'name' was missing in request",
		}).Error("failed to parse client 'name' parameter")

		return
	}

	clientPassWord := r.Form.Get("password")
	if clientPassWord == "" {
		response := ClientSignInResponse{
			Success:  false,
			ClientID: 0,
			Error:    "Expected parameter 'password' in request",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func": "SignIn",
			"err":  "form parameter 'password' was missing in request",
		}).Error("failed to parse client 'password' parameter")
		return
	}

	clientID, err := dal.SignInClient(api.db, clientName, clientPassWord)
	if err != nil {
		if clientID == 0 {
			// There wasn't any client name in the database that matched that name
			response := ClientSignInResponse{
				Success:  false,
				ClientID: 0,
				Error:    "No client has the requested name: " + clientName,
			}
			api.respondWithJSON(w, http.StatusBadRequest, response)

			return
		}

		// If the clientID wasn't found, that means the password didn't match
		response := ClientSignInResponse{
			Success:  false,
			ClientID: clientID,
			Error:    "Incorrect password",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		return
	}

	// Send a success message back
	response := ClientSignInResponse{
		Success:  true,
		ClientID: clientID,
	}
	api.respondWithJSON(w, http.StatusBadRequest, response)
}

// Private Functions

func (api *Api) sendAuthorizationResult(w http.ResponseWriter, r *http.Request, userId int, Callback string) {
	// Add the url parameters to the callback url
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
	// Check what kind of service this user is being created for
	switch Oauth2Params.PlatformName {
	case "fitbit":
		// Before we create the user, check the id to see if its in the database
		userID, err := dal.GetUserByPlatformID(db, Oauth2Params.PlatformID, Oauth2Params.PlatformName)
		if err != nil {
			return 0, err
		}

		if userID != 0 {
			// This user already exists in the Marathon User table.
			// However, the user may not exist in the clients userbase.
			// Check if they do.
			userID, err := dal.GetUserInUserbase(db, userID, Oauth2Params.ClientID)
			if err != nil {
				return 0, err
			}

			if userID == 0 {
				// The user exists, but is not in the clients userbase. Add it.
				err := dal.AddUserToUserbase(db, userID, Oauth2Params.ClientID)
				if err != nil {
					return 0, err
				}

				return userID, nil
			}

			// The user already exists both in Marathon, and in the client's userbase.
			// What are we doing here? It's over. Go home.
			return userID, nil
		}

		// Create the credentials for the user
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
		}

		log.WithFields(logrus.Fields{
			"userID": userID,
		}).Info("new fitbit user created")

		return userID, nil
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

func (api *Api) respondWithError(w http.ResponseWriter, code int, message string) {
	err := api.respondWithJSON(w, code, map[string]string{"error": message})
	if err == nil {
		api.log.WithFields(logrus.Fields{
			"err":  message,
			"code": code,
		}).Info("sent response to client")
	}
}

func (api *Api) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err := w.Write(response)

	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to send response to client")
	}

	return err
}

func checkMarathonURL(log *logrus.Logger, next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHeader := r.Header.Get("Origin")

		if originHeader != allowedOrigin {
			log.WithFields(logrus.Fields{
				"host_origin": originHeader,
			}).Warn("received request from non-allowed host")

			payload := map[string]string{"error": "Unauthorized host"}
			response, _ := json.Marshal(payload)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write(response)

			if err != nil {
				log.WithFields(logrus.Fields{
					"err": err.Error(),
				}).Error("error when trying to send response back to request sender")
			}

			return
		}

		// Call was made from Marathon Website, call the next middleware
		next.ServeHTTP(w, r)
	})
}
