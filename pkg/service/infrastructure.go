package service

/*
	File: infrastructure.go
	Last Updated: January 23, 2020
	Updated by: Manuel Poppe Richter
	Purpose: Defines the interface, functions and parameters for our web service
*/

import (
	"context"
	"errors"
)

// The service for our application
// TODO : Find out what the final endpoints and their parameters will be. These are just basic mock-ups
type MarathonService interface {
	GetCalories(ctx context.Context, userId int) (int, error)
	GetSteps(ctx context.Context, userId int) (int, error)
}

// TODO: make a better name
// the actual Structure that is going to use the MarathonService Interface
type MarathonWebService struct{}

// The implementation of the above functions
func (MarathonWebService) GetCalories(ctx context.Context, userId int) (int, error) {
	return 5, nil
}

// The implementation of the above functions
func (MarathonWebService) GetSteps(ctx context.Context, userId int) (int, error) {
	return 5, nil
}

// Error for no valid user id
var ErrNoUser = errors.New("No user exists withe the passed in ID")
