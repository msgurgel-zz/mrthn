/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

import (
	"github.com/msgurgel/marathon/pkg/model"
)

type GetUserStepsResponse struct {
	ID    int                 `json:"id,omitempty"`
	Steps []model.ValueResult `json:"steps,omitempty"`
}

type GetUserCaloriesResponse struct {
	ID       int                 `json:"id,omitempty"`
	Calories []model.ValueResult `json:"calories,omitempty"`
}

type GetUserDistanceResponse struct {
	ID       int                 `json:"id,omitempty"`
	Distance []model.ValueResult `json:"distance,omitempty"`
}

type ClientSignUpResponse struct {
	Success    bool   `json:"success"`
	ClientID   int    `json:"clientID"`
	ClientName string `json:"clientName"`
	Error      string `json:"error,omitempty"`
}

type ClientSignInResponse struct {
	Success  bool   `json:"success"`
	ClientID int    `json:"userID,omitempty"`
	Error    string `json:"error,omitempty"`
}

type CallbackUpdateResponse struct {
	Success         bool   `json:"success"`
	Error           string `json:"error,omitempty"`
	UpdatedCallback string `json:"updatedCallback,omitempty"`
}
