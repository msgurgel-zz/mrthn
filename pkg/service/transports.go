package service

/*
	File:    transports.go
	Date:    January 23, 2020
	Author:  Manuel Poppe Richter
	Purpose: Defines the accepted http requests and responses for our web service
*/

import (
	"context"
	"encoding/json"
	"net/http"
)

// the requests and responses for the GetCalories function
type GetCaloriesRequest struct {
	UserId int `json:"userId"`
}

type GetCaloriesResponse struct {
	Calories int    `json:"calories"`
	Err      string `json:"err,omitempty"` // we don't want to send an error unless something went wrong
}

type GetStepsRequest struct {
	UserId int `json:"userId"`
}

type GetStepsResponse struct {
	Steps int    `json:"steps"`
	Err   string `json:"err,omitempty"` // we don't want to send an error unless something went wrong
}

// The decoders are necessary for converting an http request into the appropriate request struct
// if there is an issue converting the https request to the appropriate struct, an error is created
// TODO: These functions look very similar, can probably abstract it into one function? This will probably be more useful the more functions we create

// DecodeGetCaloriesRequest returns the request structure for a GetCalories request, or an error if the proper arguments weren't sent
func DecodeGetCaloriesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GetCaloriesRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}

	return req, err
}

// DecodeGetStepsRequest returns the request structure for a GetSteps request, or an error if the proper arguments weren't sent
func DecodeGetStepsRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GetStepsRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	return req, err
}

// Creates the responses for our methods
func EncodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}
