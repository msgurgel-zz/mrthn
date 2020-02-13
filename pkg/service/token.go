package service

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/sirupsen/logrus"
)

type marathonClaims struct {
	ClientId int `json:"client_id"`
	jwt.StandardClaims
}

type parseToken struct {
	clientId int
	valid    bool
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

func validateJWT(db *sql.DB, tokenString string) (parseToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &marathonClaims{}, func(token *jwt.Token) (interface{}, error) {
		if claims, ok := token.Claims.(*marathonClaims); ok {
			return GetClientSecret(db, claims.ClientId)
		} else {
			return nil, errors.New("unable to parse JWT claims")
		}
	})

	if err != nil {
		return parseToken{valid: false}, err
	}

	if token.Valid {
		claims, _ := token.Claims.(*marathonClaims)
		return parseToken{
			clientId: claims.ClientId,
			valid:    true,
		}, nil
	}

	return parseToken{valid: false}, nil

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

		parseToken, err := validateJWT(db, token)
		if parseToken.valid {
			context.Set(r, "client_id", parseToken.clientId)
			next.ServeHTTP(w, r)
		} else {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("JWT token was invalid")

			respondWithError(w, http.StatusUnauthorized, "invalid JWT token")
		}
	})
}
