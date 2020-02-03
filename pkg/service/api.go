/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"

	"github.com/dgrijalva/jwt-go"
	"github.com/msgurgel/revival/pkg/model"
)

type Api struct {
	logger     *logrus.Logger
	signingKey []byte
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

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response) // TODO: deal with possible error
}
