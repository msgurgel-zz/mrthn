package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/msgurgel/marathon/pkg/environment"

	"golang.org/x/oauth2/endpoints"

	"golang.org/x/oauth2"
)

// We need to have a client that can be used to independently make calls to different apis

type Oauth2 struct {
	RequestClient *http.Client              // The client that methods can use to make the requests
	Configs       map[string]*oauth2.Config // Map of strings to OAuth Configs
	CurrentStates map[string]StateKeys
}

type OAuth2Result struct {
	AccessToken  string
	RefreshToken string
	ClientID     int
	PlatformName string
	Callback     string
	PlatformID   string
}

// When a user needs to request OAuth2 authorization, we need to save the important information in the state object
// When the callback occurs, we compare the StateObject with the one that we got back
type StateKeys struct {
	UserID   int
	Platform string
	State    []byte
	URL      string
	Callback string
	ClientID int
}

func initializeOAuth2Map(configs *environment.MarathonConfig) map[string]*oauth2.Config {
	OauthConfigs := make(map[string]*oauth2.Config)

	// Initialize all platforms OAuth2 configs
	OauthConfigs["fitbit"] = &oauth2.Config{
		RedirectURL:  configs.Callback,
		ClientID:     configs.FitBit.ClientID,
		ClientSecret: configs.FitBit.ClientSecret,
		Scopes:       []string{"activity", "profile", "settings", "heartrate"},
		Endpoint:     endpoints.Fitbit,
	}

	return OauthConfigs
}

func createStateString(service string) string {
	serviceBytes := []byte(service)

	data := make([]byte, 30) // 30 characters should be a good random string
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return "" // TODO: return an error here, log it and return an error for the end user
	}

	// add the service type to the front and the userID in the back
	stateString := append(serviceBytes, data...)

	return base64.StdEncoding.EncodeToString(stateString)
}

func NewOAuth2(configs *environment.MarathonConfig) Oauth2 {
	requestClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	return Oauth2{
		RequestClient: requestClient,
		Configs:       initializeOAuth2Map(configs),
		CurrentStates: make(map[string]StateKeys),
	}
}

func (o *Oauth2) retrieveStateObject(stateKey string) (StateKeys, error) {
	// Check if the StateKeys structure actually exists
	if State, ok := o.CurrentStates[stateKey]; ok {
		// Return the state key while removing it from the list
		delete(o.CurrentStates, stateKey)
		return State, nil
	} else {
		return StateKeys{}, errors.New("request unexpected, does not match any known authorization request")
	}
}

// ObtainUserTokens checks if the inputted state exists. If so, it attempts to exchange the passed in code for the access and refresh tokens
func (o *Oauth2) ObtainUserTokens(stateKey string, code string) (OAuth2Result, error) {

	// first things first, does this state actually exist?
	ReturnedState, err := o.retrieveStateObject(stateKey)

	if err == nil {
		// This was an expected request.
		// depending on what service was called, exchanging the code for the tokens may work slightly differently
		switch ReturnedState.Platform {
		case "fitbit":
			// exchange the code received for an access and refresh token
			token, err := o.Configs["fitbit"].Exchange(context.Background(), code)

			if err != nil {
				return OAuth2Result{Callback: ReturnedState.Callback}, err
			} else {

				// return the tokens! If we need more values, such as the expiry date, we can return more here
				return OAuth2Result{
					AccessToken:  token.AccessToken,
					RefreshToken: token.RefreshToken,
					ClientID:     ReturnedState.ClientID,
					PlatformName: ReturnedState.Platform,
					Callback:     ReturnedState.Callback,
					PlatformID:   token.Extra("user_id").(string),
				}, err

			}
		default:
			return OAuth2Result{Callback: ReturnedState.Callback}, errors.New(ReturnedState.Platform + " service does not exist")
		}
	} else {
		// this was an unexpected state
		return OAuth2Result{Callback: ReturnedState.Callback}, err
	}
}

// CreateState creates a state string that we send along with the OAuth2 request
func (o *Oauth2) CreateStateObject(callbackURL string, service string, clientID int) (StateKeys, error) {
	ReturnedKeys := StateKeys{}

	// check if the service actually exists

	// get the type of service that the user wishes to login with
	if serviceConfig, ok := o.Configs[service]; ok {
		ReturnedKeys.Platform = service

		// the user ID and service is valid, create a state string for it
		stateString := createStateString(service)

		ReturnedKeys.State = []byte(stateString)
		ReturnedKeys.URL = serviceConfig.AuthCodeURL(stateString)
		ReturnedKeys.Callback = callbackURL
		ReturnedKeys.ClientID = clientID

		// Add this state to the state map
		o.CurrentStates[string(ReturnedKeys.State)] = ReturnedKeys
	}

	return ReturnedKeys, nil
}
