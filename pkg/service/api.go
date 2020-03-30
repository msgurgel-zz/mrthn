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
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/msgurgel/marathon/pkg/auth"
	"github.com/msgurgel/marathon/pkg/dal"
	"github.com/msgurgel/marathon/pkg/helpers"
	"github.com/msgurgel/marathon/pkg/model"
	"github.com/msgurgel/marathon/pkg/platform"
)

type Api struct {
	log         *logrus.Logger
	authMethods auth.Types
	db          *sql.DB
}

func NewApi(db *sql.DB, logger *logrus.Logger, authTypes auth.Types) Api {
	return Api{
		log:         logger,
		db:          db,
		authMethods: authTypes,
	}
}

func (api *Api) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "API is working ✌️")
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
	api.getCalories(w, r, false)
}

func (api *Api) GetLargestUserCalories(w http.ResponseWriter, r *http.Request) {
	api.getCalories(w, r, true)
}

func (api *Api) getCalories(w http.ResponseWriter, r *http.Request, onlyReturnLargestValue bool) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func": "GetUserCalories"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
	}

	if !api.clientCanQueryUser(w, r, userID) {
		return
	}

	caloriesValues, err := model.GetUserCalories(api.db, api.log, userID, date, onlyReturnLargestValue)
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
	api.getDistance(w, r, false)
}

func (api *Api) GetLargestDistance(w http.ResponseWriter, r *http.Request) {
	api.getDistance(w, r, true)
}

func (api *Api) getDistance(w http.ResponseWriter, r *http.Request, onlyReturnLargestValue bool) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func": "GetUserDistance"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
	}

	if !api.clientCanQueryUser(w, r, userID) {
		return
	}

	distanceValues, err := model.GetUserDistance(api.db, api.log, userID, date, onlyReturnLargestValue)
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
	api.getSteps(w, r, false)
}

func (api *Api) GetLargestSteps(w http.ResponseWriter, r *http.Request) {
	api.getSteps(w, r, true)
}

func (api *Api) getSteps(w http.ResponseWriter, r *http.Request, onlyReturnLargestValue bool) {
	userID, date, err := api.getRequestParams(r, logrus.Fields{"func:": "GetUserSteps"})
	if err != nil {
		api.respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !api.clientCanQueryUser(w, r, userID) {
		return
	}

	stepsValues, err := model.GetUserSteps(api.db, api.log, userID, date, onlyReturnLargestValue)
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

	// Token exists; validate it
	parseToken, err := validateJWT(api.db, token[0])
	if err != nil || !parseToken.valid {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("JWT was missing or invalid")

		api.respondWithError(w, http.StatusUnauthorized, "Access token is missing or invalid")
		return
	}

	// Start populating params struct
	params := auth.CreateStateObjectParams{ClientID: parseToken.clientID}

	// Get the other URL params
	service, serviceOk := r.URL.Query()["service"]
	callbackURL, callbackOk := r.URL.Query()["callback"]
	userIDStrings, userIDOk := r.URL.Query()["userID"] // Optional param. Used when adding a new platform account to existing user

	// Validate service param in URL
	if !serviceOk || len(service) != 1 {
		api.log.WithFields(logrus.Fields{
			"func":     "Login",
			"clientID": parseToken.clientID,
		}).Error("missing URL param 'service'")

		api.respondWithError(w, http.StatusBadRequest,
			"expected single 'service' parameter with name of service to authenticate with",
		)

		return
	}
	if !platform.IsPlatformAvailable(service[0]) {
		// Invalid platform was passed
		api.log.WithFields(logrus.Fields{
			"func":     "Login",
			"clientID": parseToken.clientID,
			"service":  service,
		}).Error("invalid service was given")

		api.respondWithError(w, http.StatusBadRequest, "invalid service. accepted are 'google' and 'fitbit'")

		return
	}

	// Validate callback param in URL
	if !callbackOk || len(callbackURL) != 1 {
		api.log.WithFields(logrus.Fields{
			"func":     "Login",
			"clientID": parseToken.clientID,
		}).Error("missing URL param 'callback'")

		api.respondWithError(w, http.StatusBadRequest,
			"expected single 'callback' parameter to contain valid callback url")

		return
	}

	// Add validated service and callback to the params struct
	params.Service = service[0]
	params.CallbackURL = callbackURL[0]

	// Check if the optional parameter userID was given
	if userIDOk {
		// Check if there's only one
		if len(userIDStrings) != 1 {
			api.log.WithFields(logrus.Fields{
				"func":     "Login",
				"clientID": parseToken.clientID,
				"userID":   userIDStrings,
			}).Error("more than one userID param was given")

			api.respondWithError(w, http.StatusBadRequest,
				"more than one optional parameter 'userID' was passed")

			return
		}

		// Check if is a number
		userID, err := strconv.Atoi(userIDStrings[0])
		if err != nil {
			api.log.WithFields(logrus.Fields{
				"func":     "Login",
				"clientID": parseToken.clientID,
				"userID":   userIDStrings,
			}).Error("userID was not a number")

			api.respondWithError(w, http.StatusBadRequest,
				"optional parameter 'userID' was expected to be a number")

			return
		}

		// Check if client has access to the specified user
		hasAccess := api.clientHasAccessToUser(w, parseToken.clientID, userID)
		if !hasAccess {
			return
		}

		// Check if user already has linked account of the given platform
		userPlatforms, err := dal.GetPlatformNames(api.db, userID)
		if err != nil {
			api.log.WithFields(logrus.Fields{
				"func": "Login",
				"user": userID,
				"err":  err,
			}).Error("failed to get platform of user from db")
			api.respondWithError(w, http.StatusInternalServerError, "something went wrong, try again later.")

			return
		}

		var found bool
		for _, p := range userPlatforms {
			if p == service[0] {
				found = true
				break
			}
		}

		if found {
			// User already has an account linked from the given platform
			api.log.WithFields(logrus.Fields{
				"func":     "Login",
				"platform": service[0],
				"client":   parseToken.clientID,
				"user":     userID,
			}).Warn("client tried to link already linked platform account")

			api.respondWithError(w, http.StatusBadRequest, "user has already linked account of service '"+service[0]+"'")

			return
		}

		// Add the validated userID to the params struct
		params.UserID = userID
	}

	// TODO: This is dependent on OAuth2. When new auth types are needed, this will have to be changed
	requestStateObject, ok := api.authMethods.Oauth2.CreateStateObject(params)
	if ok == nil {
		url := requestStateObject.URL                          // Check what type of request was made using the StateObject
		http.Redirect(w, r, url, http.StatusTemporaryRedirect) // Redirect with the stateObjects url
	}
}

func (api *Api) Callback(w http.ResponseWriter, r *http.Request) {
	// TODO: Remove dependency on OAuth2
	// Check that the state returned was valid
	Oauth2Result, callback, err := api.authMethods.Oauth2.ObtainUserTokens(
		r.FormValue("state"),
		r.FormValue("code"),
	)
	if err != nil {
		// Something went wrong. Instead of the result, send back the error
		api.log.WithFields(logrus.Fields{
			"func":  "Callback",
			"err":   err,
			"state": r.FormValue("state"),
		}).Error("failed to retrieve OAuth2 token for user")

		if callback != "" {
			api.sendAuthorizationResult(w, r, 0, callback)
		}

		return
	}

	// Is this request for a new user or an existing user?
	if Oauth2Result.UserID == 0 {
		// New user
		userID, err := api.createUser(&Oauth2Result)
		if err != nil {
			api.log.WithFields(logrus.Fields{
				"func": "Callback",
				"err":  err,
			}).Error("failed to create a new user in the database")
			api.sendFailedAuthorizationResult(w, r, callback)

			return
		}

		api.sendAuthorizationResult(w, r, userID, callback)
	} else {
		// Existing user
		err = api.createUserCredentials(&Oauth2Result, Oauth2Result.UserID)
		if err != nil {
			api.log.WithFields(logrus.Fields{
				"func":   "Callback",
				"userID": Oauth2Result.UserID,
				"err":    err,
			}).Error("failed to add new credentials to existing user")
			api.sendFailedAuthorizationResult(w, r, callback)

			return
		}

		api.sendAuthorizationResult(w, r, Oauth2Result.UserID, callback)
	}
}

func (api *Api) SignUp(w http.ResponseWriter, r *http.Request) {
	// get the new values of the client
	err := r.ParseMultipartForm(500)

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
	clientID, err := dal.GetClientID(api.db, clientName)
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

	if clientID != 0 {
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

	clientPassword := r.Form.Get("password")

	if clientPassword == "" {
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

	newClientID, err := dal.InsertNewClient(api.db, clientName, clientPassword)
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
		ClientID:   newClientID,
		ClientName: clientName,
	}
	api.respondWithJSON(w, http.StatusOK, response)
}

func (api *Api) SignIn(w http.ResponseWriter, r *http.Request) {
	// TODO: Make sure that this uses ParseForm instead of ParseMultipartForm
	err := r.ParseMultipartForm(500)
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

func (api *Api) UpdateClientCallback(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "Error occurred while attempting to parse form values",
		}
		api.respondWithJSON(w, http.StatusInternalServerError, response)

		api.log.WithFields(logrus.Fields{
			"func": "UpdateClientCallback",
			"err":  err,
		}).Error("failed to parse request form values")

		return
	}

	newCallback := r.Form.Get("callback")
	if newCallback == "" {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "Expected parameter 'name' in request",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func": "UpdateClientCallback",
			"err":  "form parameter 'callback' was missing in request",
		}).Error("failed to parse client 'callback' parameter")

		return
	}

	vars := mux.Vars(r)
	if vars["clientID"] == "" {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "missing clientID",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func": "UpdateClientCallback",
			"err":  "clientID not received",
		}).Error("failed to parse client 'clientID' parameter")
	}

	clientID, err := strconv.Atoi(vars["clientID"])
	if err != nil {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "clientID must be an integer",
		}
		api.respondWithJSON(w, http.StatusBadRequest, response)

		api.log.WithFields(logrus.Fields{
			"func":     "UpdateClientCallback",
			"err":      "clientID received must be an integer",
			"received": vars["clientID"],
		}).Error("failed to parse client 'clientID' parameter")

		return
	}

	// We have the new callback so now update the client with it
	result, err := dal.UpdateCallback(api.db, clientID, newCallback)
	if err != nil {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "error occurred while updating client callback",
		}
		api.respondWithJSON(w, http.StatusInternalServerError, response)

		api.log.WithFields(logrus.Fields{
			"func": "UpdateClientCallback",
			"err":  err,
		}).Error("failed to update client callback")

		return
	}

	if !result {
		response := CallbackUpdateResponse{
			Success: false,
			Error:   "clientID does not match any registered client",
		}

		api.respondWithJSON(w, http.StatusBadRequest, response)
		return
	}

	response := CallbackUpdateResponse{
		Success:         true,
		UpdatedCallback: newCallback,
	}
	api.respondWithJSON(w, http.StatusInternalServerError, response)
}

// Private Functions

func (api *Api) sendAuthorizationResult(w http.ResponseWriter, r *http.Request, userId int, Callback string) {
	// Add the url parameters to the callback url
	Callback += fmt.Sprintf("?userId=%d", userId)

	api.log.WithFields(logrus.Fields{
		"callback": Callback,
		"userId":   userId,
	}).Info("sending login result to client")

	http.Redirect(w, r, Callback, http.StatusTemporaryRedirect)
}

func (api *Api) sendFailedAuthorizationResult(w http.ResponseWriter, r *http.Request, Callback string) {
	api.log.WithFields(logrus.Fields{
		"callback": Callback,
	}).Info("sending failed login result to client")

	http.Redirect(w, r, Callback, http.StatusInternalServerError)
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

func (api *Api) clientHasAccessToUser(w http.ResponseWriter, clientID, userID int) bool {
	dbUser, err := dal.GetUserInUserbase(api.db, userID, clientID)
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err": err,
		}).Error("failed to get user from the database")

		api.respondWithError(w, http.StatusInternalServerError, "Something went wrong... Try again later")

		return false
	}

	if dbUser != userID {
		// Client does not have permission to access this user!
		api.log.WithFields(logrus.Fields{
			"userID":   userID,
			"clientID": clientID,
		}).Warn("client tried to access unauthorized or non-existent user")

		api.respondWithError(w, http.StatusNotFound, "User with specified ID was not found")

		return false
	}

	return true
}

func (api *Api) clientCanQueryUser(w http.ResponseWriter, r *http.Request, userID int) bool {
	clientID := context.Get(r, "client_id") // This was set during JWT validation middleware
	if clientID == nil {
		api.log.Error("failed to get client ID from JWT token")
		api.respondWithError(w, http.StatusInternalServerError, "Something went wrong... Try again later")

		return false
	}

	return api.clientHasAccessToUser(w, clientID.(int), userID)
}

func (api *Api) createUser(Oauth2Params *auth.OAuth2Result) (int, error) {
	// Before we create the user, check the ID to see if it's in the database
	userID, err := dal.GetUserByPlatformID(api.db, Oauth2Params.PlatformID, Oauth2Params.PlatformName)
	if err != nil {
		return 0, err
	}

	if userID != 0 {
		// This user already exists in the Marathon User table.
		// However, the user may not exist in the clients userbase.
		// Check if they do.
		userID, err := dal.GetUserInUserbase(api.db, userID, Oauth2Params.ClientID)
		if err != nil {
			return 0, err
		}

		if userID == 0 {
			// The user exists, but is not in the clients userbase. Add it.
			err := dal.InsertUserToUserbase(api.db, userID, Oauth2Params.ClientID)
			if err != nil {
				return 0, err
			}

			return userID, nil
		}

		// The user already exists both in Marathon, and in the client's userbase.
		// What are we doing here? It's over. Go home.
		return userID, nil
	}

	// Create user
	newUserID, err := dal.InsertNewUser(api.db)

	// Create the credentials for the user
	err = api.createUserCredentials(Oauth2Params, newUserID)
	if err != nil {
		return 0, err
	}

	api.log.WithFields(logrus.Fields{
		"userID":   userID,
		"platform": Oauth2Params.PlatformName,
	}).Info("new user created")

	return newUserID, nil
}

func (api *Api) createUserCredentials(Oauth2Params *auth.OAuth2Result, userID int) error {
	var connectionParams = []string{
		"oauth2",
		Oauth2Params.Token.TokenType,
		Oauth2Params.Token.Expiry.Format(helpers.ISO8601Layout),
		Oauth2Params.Token.AccessToken,
		Oauth2Params.Token.RefreshToken,
	}
	connStr, err := helpers.FormatConnectionString(connectionParams)
	if err != nil {
		return err
	}

	params := dal.CredentialParams{
		UserID:           userID,
		ClientID:         Oauth2Params.ClientID,
		PlatformName:     Oauth2Params.PlatformName,
		UPID:             Oauth2Params.PlatformID,
		ConnectionString: connStr,
	}
	userID, err = dal.InsertUserCredentials(api.db, params)
	if err != nil {
		api.log.WithFields(logrus.Fields{
			"err":      err,
			"platform": Oauth2Params.PlatformName,
		}).Error("failed to create a new user")

		return err
	}
	return nil
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
