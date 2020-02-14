/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"database/sql"
	"net/http"

	"github.com/msgurgel/marathon/pkg/environment"

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

func NewRouter(db *sql.DB, logger *logrus.Logger, config *environment.MarathonConfig) *mux.Router {
	routes := prepareRoutes(db, logger, config)
	router := mux.NewRouter().StrictSlash(true)

	// Initialize routes
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		if route.Secure {
			handler = jwtMiddleware(db, logger, handler)
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

func prepareRoutes(db *sql.DB, logger *logrus.Logger, config *environment.MarathonConfig) Routes {
	api := NewApi(db, logger, config)

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

		Route{
			"Login",
			"GET",
			"/login",
			true,
			api.Login,
		},

		Route{
			"Callback",
			"GET",
			"/callback",
			false,
			api.Callback,
		},
	}

	return routes
}
