/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	Secure      bool
	HandlerFunc http.HandlerFunc
}

type Routes []Route

func NewRouter(logger *logrus.Logger, secret string) *mux.Router {
	routes := prepareRoutes(logger, secret)
	router := mux.NewRouter().StrictSlash(true)

	// Setup JWT middleware for secure endpoints
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	// Initialize routes
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		if route.Secure {
			handler = jwtMiddleware.Handler(handler)
		}

		handler = Logger(logger, handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

func prepareRoutes(logger *logrus.Logger, secret string) Routes {
	api := Api{}
	api.logger = logger
	api.signingKey = []byte(secret)

	routes := Routes{
		Route{
			"Index",
			"GET",
			"/",
			false,
			api.Index,
		},

		Route{
			"GetToken",
			"GET",
			"/get-token",
			false,
			api.GetToken,
		},

		Route{
			"GetUserCalories",
			"GET",
			"/user/{userID}/calories",
			true,
			api.GetUserCalories,
		},

		Route{
			"GetUserSteps",
			"GET",
			"/user/{userID}/steps",
			true,
			api.GetUserSteps,
		},
	}

	return routes
}
