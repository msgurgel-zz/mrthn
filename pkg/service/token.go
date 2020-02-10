package service

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

type marathonClaims struct {
	ClientId int `json:"client_id"`
	jwt.StandardClaims
}

func generateJWT(clientId int, secret []byte) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["client_id"] = clientId
	// claims["exp"] = time.Now().Add(time.Hour * 24).Unix() TODO: Add exp back

	// Sign the token with the given secret
	tokenString, _ := token.SignedString(secret)
	return tokenString, nil
}

func validateJWT(db *sql.DB, tokenString string) (bool, error) {
	token, err := jwt.ParseWithClaims(tokenString, &marathonClaims{}, func(token *jwt.Token) (interface{}, error) {
		if claims, ok := token.Claims.(*marathonClaims); ok {
			return GetClientSecret(db, claims.ClientId)
		} else {
			return nil, errors.New("unable to parse JWT claims")
		}
	})

	if err != nil {
		return false, err
	}

	return token.Valid, nil
}

func jwtMiddleware(db *sql.DB, log *logrus.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string

		// Get token from the Authorization header
		// format: Authorization: Bearer
		tokens, ok := r.Header["Authorization"]
		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}

		valid, err := validateJWT(db, token)
		if valid {
			next.ServeHTTP(w, r)
		} else {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("JWT token was invalid")

			respondWithError(w, http.StatusUnauthorized, "invalid JWT token")
		}
	})
}
