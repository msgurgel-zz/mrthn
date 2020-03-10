/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import "github.com/msgurgel/marathon/pkg/model"

type GetUserStepsResponse200 struct {
	ID int `json:"id,omitempty"`

	Steps []model.ValueResult `json:"steps,omitempty"`
}

type GetUserCaloriesResponse200 struct {
	ID int `json:"id,omitempty"`

	Calories []model.ValueResult `json:"steps,omitempty"`
}

type GetUserDistanceResponse200 struct {
	ID int `json:"id,omitempty"`

	Distance []model.ValueResult `json:"distance,omitempty"`
}

type ClientSignUpResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type ClientSignInResponse struct {
	Success  bool   `json:"success"`
	ClientId int    `json:"userID,omitempty"`
	Error    string `json:"error,omitempty"`
}
