package auth

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// GetFitbitOauth2Config returns an OAuth Config object specific for Fitbit
func GetFitbitOauth2Config() *oauth2.Config {

	FitBitOAuthConfig := &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     "FAKE_CLIENT_ID",
		ClientSecret: "FAKE_CLIENT_SECRET",
		Scopes:       []string{"activity", "profile", "settings", "heartrate"},
		Endpoint:     endpoints.Fitbit,
	}

	return FitBitOAuthConfig
}
