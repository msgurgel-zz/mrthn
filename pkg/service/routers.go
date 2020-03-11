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
	Name                string
	Method              string
	Pattern             string
	Secure              bool
	HandlerFunc         http.HandlerFunc
	OnlyMarathonRequest bool
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
		if route.OnlyMarathonRequest {
			handler = checkOrigin(logger, handler, config.MarathonURL)
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
			false,
		},

		Route{
			"GetToken",
			"GET",
			"/get-token",
			false,
			api.GetToken,
			true,
		},

		Route{
			"GetUserCalories",
			"GET",
			"/user/{userID}/calories",
			true,
			api.GetUserCalories,
			false,
		},

		Route{
			"GetUserSteps",
			"GET",
			"/user/{userID}/steps",
			true,
			api.GetUserSteps,
			false,
		},

		Route{
			"GetUserDistance",
			"GET",
			"/user/{userID}/distance",
			true,
			api.GetUserDistance,
			false,
		},

		Route{
			"Login",
			"GET",
			"/login",
			false,
			api.Login,
			true,
		},

		Route{
			"Callback",
			"GET",
			"/callback",
			false,
			api.Callback,
			false,
		},

		Route{
			"SignUp",
			"POST",
			"/signup",
			false,
			api.SignUp,
			true,
		},
		Route{
			"SignIn",
			"POST",
			"/signin",
			false,
			api.SignIn,
			true,
		},
	}

	return routes
}
