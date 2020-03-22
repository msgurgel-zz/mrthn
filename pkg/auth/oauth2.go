package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/msgurgel/marathon/pkg/environment"
	"golang.org/x/oauth2/endpoints"

	"golang.org/x/oauth2"
)

// We need to have a client that can be used to independently make calls to different APIs

type OAuth2 struct {
	RequestClient *http.Client              // The client that methods can use to make the requests
	Configs       map[string]*oauth2.Config // Map of strings to OAuth Configs
	CurrentStates map[string]StateKeys
}

type OAuth2Result struct {
	Token        *oauth2.Token
	ClientID     int
	PlatformName string
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

// UserProfileResponse is a json structure representing the response of calling the users google profile
// Only used when creating a new user and switching the tokens
type UserProfileResponse struct {
	EmailAddress  string `json:"emailAddress"`
	MessagesTotal int    `json:"messagesTotal,omitempty"`
	ThreadsTotal  int    `json:"threadsTotal,omitempty"`
	HistoryID     string `json:"historyId,omitempty"`
}

func NewOAuth2(configs *environment.MarathonConfig) OAuth2 {
	requestClient := &http.Client{
		Timeout: 10 * time.Second, // TODO: Make this an environment variable
	}

	return OAuth2{
		RequestClient: requestClient,
		Configs:       initializeOAuth2Map(configs),
		CurrentStates: make(map[string]StateKeys),
	}
}

func (o *OAuth2) retrieveStateObject(stateKey string) (StateKeys, error) {
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
func (o *OAuth2) ObtainUserTokens(stateKey string, code string) (result OAuth2Result, callback string, err error) {
	// First things first, does this state actually exist?
	returnedState, err := o.retrieveStateObject(stateKey)
	if err != nil {
		// This was an unexpected state
		return OAuth2Result{}, "", err
	}

	// This was an expected request.
	// Depending on what service was called, exchanging the code for the tokens may work slightly differently
	switch returnedState.Platform {
	case "fitbit":
		// exchange the code received for an access and refresh token
		token, err := o.Configs["fitbit"].Exchange(context.Background(), code)

		if err != nil {
			return OAuth2Result{}, returnedState.Callback, err
		} else {
			// Return the tokens! If we need more values, such as the expiry date, we can return more here
			result = OAuth2Result{
				Token:        token,
				ClientID:     returnedState.ClientID,
				PlatformName: returnedState.Platform,
				PlatformID:   token.Extra("user_id").(string),
			}

			return result, returnedState.Callback, nil
		}
	case "google":
		// Exchange the code received for an access and refresh token
		tokens, err := o.Configs["google"].Exchange(context.Background(), code)
		if err != nil {
			return OAuth2Result{}, returnedState.Callback, err
		}

		// Prepare results
		googleOauth2Result := OAuth2Result{
			Token:        tokens,
			ClientID:     returnedState.ClientID,
			PlatformName: returnedState.Platform,
		}

		// Before we can return the result, we need to find out the users email address
		client := o.Configs["google"].Client(context.Background(), tokens)
		resp, err := client.Get("https://www.googleapis.com/gmail/v1/users/me/profile")
		if err != nil {
			return OAuth2Result{}, returnedState.Callback, err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return OAuth2Result{}, returnedState.Callback, err
		}

		// Unmarshal the JSON response into a google user profile response
		userProfile := UserProfileResponse{}
		err = json.Unmarshal(body, &userProfile)
		if err != nil {
			return OAuth2Result{}, returnedState.Callback, err
		}

		// Put the users email into the oauth2 result
		googleOauth2Result.PlatformID = userProfile.EmailAddress

		return googleOauth2Result, returnedState.Callback, nil

	default:
		return OAuth2Result{}, returnedState.Callback, errors.New(returnedState.Platform + " service does not exist")
	}
}

// CreateState creates a state string that we send along with the OAuth2 request
func (o *OAuth2) CreateStateObject(callbackURL string, service string, clientID int) (StateKeys, error) { // TODO: make the params a Params object
	ReturnedKeys := StateKeys{}

	// Get the type of service that the user wishes to login with
	if serviceConfig, ok := o.Configs[service]; ok {
		ReturnedKeys.Platform = service

		// The service is valid, create a state string for it
		stateString := createStateString(service)

		ReturnedKeys.State = []byte(stateString)

		if service == "google" {
			ReturnedKeys.URL = serviceConfig.AuthCodeURL(stateString, oauth2.AccessTypeOffline)
		} else {
			ReturnedKeys.URL = serviceConfig.AuthCodeURL(stateString)
		}

		ReturnedKeys.Callback = callbackURL
		ReturnedKeys.ClientID = clientID

		// Add this state to the state map
		o.CurrentStates[string(ReturnedKeys.State)] = ReturnedKeys
	}

	return ReturnedKeys, nil
}

func RefreshOAuth2Tokens(tokens *oauth2.Token, conf *oauth2.Config) (*oauth2.Token, error) {
	// Attempt to refresh token
	tokenSource := conf.TokenSource(context.Background(), tokens)
	newTokens, err := tokenSource.Token()
	if err != nil {
		return nil, errors.New("failed to refresh token: " + err.Error())
	}

	return newTokens, nil
}

func initializeOAuth2Map(configs *environment.MarathonConfig) map[string]*oauth2.Config {
	OAuthConfigs := make(map[string]*oauth2.Config)

	// Initialize all platforms OAuth2 configs
	OAuthConfigs["fitbit"] = &oauth2.Config{
		RedirectURL:  configs.Callback,
		ClientID:     configs.Fitbit.ClientID,
		ClientSecret: configs.Fitbit.ClientSecret,
		Scopes:       []string{"activity", "profile", "settings", "heartrate"},
		Endpoint:     endpoints.Fitbit,
	}

	OAuthConfigs["google"] = &oauth2.Config{
		RedirectURL:  configs.Callback,
		ClientID:     configs.Google.ClientID,
		ClientSecret: configs.Google.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/fitness.activity.read", "https://www.googleapis.com/auth/fitness.location.read", "https://www.googleapis.com/auth/gmail.readonly"},
		Endpoint:     endpoints.Google,
	}

	return OAuthConfigs
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
