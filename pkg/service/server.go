package service

/*
	File:    server.go
	Date:    January 23, 2020
	Author:  Manuel Poppe Richter
	Purpose: Creates a custom server interface that has the methods for our specific server on it
	Most of this code is unchanged from https://dev.to/napolux/how-to-write-a-microservice-in-go-with-go-kit-a66
*/

import (
	"context"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func NewHTTPServer(ctx context.Context, endpoints Endpoints) http.Handler {
	r := mux.NewRouter()
	//  TODO: This what in the tutorial but not explained. What is this?!?! I believe it will just add the header to every response
	r.Use(commonMiddleware) // @see https://stackoverflow.com/a/51456342

	r.Methods("POST").Path("/GetCalories").Handler(httptransport.NewServer(
		endpoints.GetCaloriesEndpoint,
		DecodeGetCaloriesRequest,
		EncodeResponse,
	))

	r.Methods("POST").Path("/GetSteps").Handler(httptransport.NewServer(
		endpoints.GetStepsEndpoint,
		DecodeGetStepsRequest,
		EncodeResponse,
	))

	return r
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
