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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/msgurgel/marathon/pkg/auth"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"

	"github.com/dgrijalva/jwt-go"
	"github.com/msgurgel/marathon/pkg/model"
)

type Api struct {
	logger      *logrus.Logger
	signingKey  []byte
	authMethods auth.Types
	// The database conn obj will be here
}

func (api *Api) Index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
}

func (api *Api) GetToken(w http.ResponseWriter, r *http.Request) {
	// Create the token
	token := jwt.New(jwt.SigningMethodHS256)

	// Create a map to store our claims
	claims := token.Claims.(jwt.MapClaims)

	// Set token claims
	claims["admin"] = true
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	// Sign the token with our secret
	tokenString, _ := token.SignedString(api.signingKey)

	// Finally, write the token to the browser window
	w.Write([]byte(tokenString))
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
		// create the state object
		RequestStateObject, ok := api.authMethods.Oauth2.CreateStateObject(callBackURL[0], service[0])

		if ok == nil {
			// check what type of request was made using the StateObject

			// redirect with the stateObjects url
			url := RequestStateObject.URL
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		}
	}

}

func (api *Api) Callback(w http.ResponseWriter, r *http.Request) {
	// first thing first, check that the state returned was valid
	AccessToken, RefreshToken, Callback, err := api.authMethods.Oauth2.ObtainUserTokens(r.FormValue("state"), r.FormValue("code"))

	if err == nil {

		// TODO: create a new user here into the database. For now, just print the access and refresh tokens
		fmt.Println(AccessToken)
		fmt.Println(RefreshToken)

		// Create the body for the callback
		var jsonStr = []byte(`{"userId":"50"}`)
		api.sendAuthorizationResult(jsonStr, Callback)

	} else {
		// something went wrong, instead of the result, send back the error
		var jsonStr = []byte(`{"error":"` + err.Error() + `"}`)
		api.sendAuthorizationResult(jsonStr, Callback)
	}

}

func (api *Api) sendAuthorizationResult(body []byte, Callback string) {
	req, _ := http.NewRequest("POST", Callback, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	api.logger.Info("sending authorization result [" + string(body) + "] to [" + Callback + "]")

	callbackResponse, err := http.Post(Callback, "application/json", bytes.NewBuffer(body))

	if err != nil {
		// log the error
		api.logger.Error(err)
		return
	}

	defer callbackResponse.Body.Close()
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
