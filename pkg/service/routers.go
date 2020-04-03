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

	"github.com/msgurgel/marathon/pkg/auth"

	"github.com/rs/cors"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Route struct {
	Name                string
	Method              string
	Pattern             string
	Secure              bool
	MarathonWebsiteOnly bool
	HandlerFunc         http.HandlerFunc
}

type Routes []Route

func NewRouter(db *sql.DB, logger *logrus.Logger, authTypes auth.Types, marathonWebsiteURL string) *mux.Router {
	routes := prepareRoutes(db, logger, authTypes)
	router := mux.NewRouter().StrictSlash(true)

	// Initialize routes
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc

		// JWT Middleware
		if route.Secure {
			handler = jwtMiddleware(db, logger, handler)
		}

		// Check Marathon Website Origin Middleware
		if route.MarathonWebsiteOnly {
			handler = checkMarathonURL(logger, handler, marathonWebsiteURL)
		}

		// CORS Middleware
		handler = cors.Default().Handler(handler)

		// Logger Middleware
		handler = Logger(logger, handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

func prepareRoutes(db *sql.DB, logger *logrus.Logger, authTypes auth.Types) Routes {
	api := NewApi(db, logger, authTypes)

	routes := Routes{
		Route{
			"Index",
			"GET",
			"/",
			false,
			false,
			api.Index,
		},

		Route{
			"GetToken",
			"GET",
			"/get-token",
			false,
			true,
			api.GetToken,
		},

		Route{
			"GetUserCalories",
			"GET",
			"/user/{userID}/calories",
			true,
			false,
			api.GetCalories,
		},

		Route{
			"GetUserSteps",
			"GET",
			"/user/{userID}/steps",
			true,
			false,
			api.GetSteps,
		},

		Route{
			"GetDistance",
			"GET",
			"/user/{userID}/distance",
			true,
			false,
			api.GetDistance,
		},

		Route{
			"Login",
			"GET",
			"/login",
			false,
			false,
			api.Login,
		},

		Route{
			"Callback",
			"GET",
			"/callback",
			false,
			false,
			api.Callback,
		},

		Route{
			"Callback",
			"POST",
			"/client/{clientID}/callback",
			false,
			true,
			api.UpdateClientCallback,
		},

		Route{
			"Callback",
			"GET",
			"/client/{clientID}/callback",
			false,
			true,
			api.GetClientCallback,
		},

		Route{
			"SignUp",
			"POST",
			"/signup",
			false,
			true,
			api.SignUp,
		},
		Route{
			"SignIn",
			"POST",
			"/signin",
			false,
			true,
			api.SignIn,
		},

		Route{
			"GetValueOverPeriod",
			"GET",
			"/user/{userID}/{resource}/over-period",
			true,
			false,
			api.GetValueOverPeriod,
		},
	}

	return routes
}
