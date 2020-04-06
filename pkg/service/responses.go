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

type GetValueResponse struct {
	ID     int                 `json:"id,omitempty"`
	Result []model.ValueResult `json:"result,omitempty"`
}

type ClientSignUpResponse struct {
	Success    bool   `json:"success"`
	ClientID   int    `json:"clientID"`
	ClientName string `json:"clientName"`
	Error      string `json:"error,omitempty"`
}

type ClientSignInResponse struct {
	Success  bool   `json:"success"`
	ClientID int    `json:"clientID,omitempty"`
	Error    string `json:"error,omitempty"`
}

type GetCallbackResponse struct {
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Callback string `json:"callback,omitempty"`
}

type CallbackUpdateResponse struct {
	Success         bool   `json:"success"`
	Error           string `json:"error,omitempty"`
	UpdatedCallback string `json:"updatedCallback,omitempty"`
}
