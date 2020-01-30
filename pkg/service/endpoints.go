package service

/*
	File:    endpoints.go
	Date:    January 23, 2020
	Author:  Manuel Poppe Richter
	Purpose: Exposes the endpoints that our web service will have,
*/

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
)

// Endpoints are exposed
type Endpoints struct {
	GetCaloriesEndpoint endpoint.Endpoint
	GetStepsEndpoint    endpoint.Endpoint
}

// MakeGetCaloriesEndpoints returns the response of the get calories endpoints
func MakeGetCaloriesEndpoint(srv MarathonService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetCaloriesRequest) // we need to get the user ID from the request
		calories, err := srv.GetCalories(ctx, req.UserId)
		if err != nil {
			return GetCaloriesResponse{calories, err.Error()}, nil
		}
		return GetCaloriesResponse{calories, ""}, nil
	}
}

// MakeGetStepsEndpoint Returns the response from our GetSteps endpoint
func MakeGetStepsEndpoint(srv MarathonService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetStepsRequest) // we need to get the user ID from the request
		steps, err := srv.GetSteps(ctx, req.UserId)
		if err != nil {
			return GetStepsResponse{steps, err.Error()}, nil
		}
		return GetStepsResponse{steps, ""}, nil
	}
}

// mapping the endpoints
// these are going to be the ones handling the requests, calling the underlying method, and creating the return response

// GetCalories maps the requests for getting Calories to the Get Calories function
func (e Endpoints) GetCalories(ctx context.Context, userId int) (int, error) {
	req := GetCaloriesRequest{UserId: userId}
	resp, err := e.GetCaloriesEndpoint(ctx, req)
	if err != nil {
		return 0, err
	}
	getCaloriesResp := resp.(GetCaloriesResponse)
	if getCaloriesResp.Err != "" {
		return 0, errors.New(getCaloriesResp.Err)
	}
	return getCaloriesResp.Calories, nil
}

//  GetSteps maps the request for getting steps to the Getting steps function
func (e Endpoints) GetSteps(ctx context.Context, userId int) (int, error) {
	req := GetStepsRequest{UserId: userId}
	resp, err := e.GetStepsEndpoint(ctx, req)
	if err != nil {
		return 0, err
	}
	getStepsResp := resp.(GetStepsResponse)
	if getStepsResp.Err != "" {
		return 0, errors.New(getStepsResp.Err)
	}
	return getStepsResp.Steps, nil
}
