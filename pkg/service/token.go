package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/sirupsen/logrus"
)

type parseToken struct {
	clientID int
	valid    bool
}

func generateJWT(clientID int, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		// TODO: Make client ID not an integer
		Audience: strconv.Itoa(clientID),
	})

	// TODO: Add expiration to the token

	// Sign the token with the given secret
	tokenString, _ := token.SignedString(secret)
	return tokenString, nil
}

func validateJWT(db *sql.DB, tokenString string) (parseToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if claims, ok := token.Claims.(*jwt.StandardClaims); ok {
			clientID, _ := strconv.Atoi(claims.Audience)
			return dal.GetClientSecret(db, clientID)
		} else {
			return nil, errors.New("unable to parse JWT claims")
		}
	})

	if err != nil {
		return parseToken{valid: false}, err
	}

	if token.Valid {
		claims, _ := token.Claims.(*jwt.StandardClaims)
		clientID, _ := strconv.Atoi(claims.Audience)
		return parseToken{
			clientID: clientID,
			valid:    true,
		}, nil
	}

	return parseToken{valid: false}, nil

}

func jwtMiddleware(db *sql.DB, log *logrus.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Get token from the Authorization header

		tokens, ok := r.Header["Authorization"]

		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		parseToken, err := validateJWT(db, token)
		if parseToken.valid {
			context.Set(r, "client_id", parseToken.clientID)
			next.ServeHTTP(w, r)
		} else {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("JWT was invalid")

			// just copy/paste the code for this part

			payload := map[string]string{"error": err.Error()}

			response, _ := json.Marshal(payload)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write(response) // TODO: deal with possible error

			if err != nil {

			}
		}
	})
}
