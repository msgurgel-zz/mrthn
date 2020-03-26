package environment

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// MarathonConfig is the overall structure that will contain our environment configs for the marathon service
type MarathonConfig struct {
	Server             serverConfig
	DBConnectionString string
	Fitbit             platformConfig
	Google             platformConfig
	Strava             platformConfig
	Callback           string        // This will be the callback for all services. If we need multiple, this may need to change
	ClientTimeout      time.Duration // The timeout for the client that is used to make requests for Marathon
	MarathonWebsiteURL string        // We will only accept client SignUp requests if it comes from the Marathon website
}

// Server config options
type serverConfig struct {
	Port         string
	ReadTimeOut  time.Duration
	WriteTimeOut time.Duration
	IdleTimeout  time.Duration
}

// Config struct specifically for Fitbit client ids, secrets, etc
type platformConfig struct {
	ClientID     string
	ClientSecret string
}

// ReadEnvFile takes the environment variables, and puts them all into an EnvironmentConfig struct
func ReadEnvFile(env string) (*MarathonConfig, error) {
	// Create the Environment Config struct we will return to the user
	setConfig := MarathonConfig{}

	if env == "development" {
		// Set environment vars using .env file
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	}

	// Get the callback for all services
	callbackUrl := os.Getenv("CALLBACK")
	if callbackUrl == "" {
		return nil, errors.New("environment variable CALLBACK is not set")
	}
	setConfig.Callback = callbackUrl

	// Get the Marathon URL
	marathonURL := os.Getenv("MARATHON_WEBSITE_URL")
	if marathonURL == "" {
		return nil, errors.New("environment variable MARATHON_WEBSITE_URL is not set")
	}
	setConfig.MarathonWebsiteURL = marathonURL

	// Get the client timeout
	clientTimeout, err := strconv.Atoi(os.Getenv("CLIENT_TIMEOUT"))
	if err != nil {
		return nil, errors.New("failed to convert CLIENT_TIMEOUT to int")
	}
	setConfig.ClientTimeout = time.Second * time.Duration(clientTimeout)

	// Start parsing the environment variables
	readTime, err := strconv.Atoi(os.Getenv("READ_TIMEOUT"))
	if err != nil {
		return nil, errors.New("failed to convert READ_TIMEOUT to int")
	}

	writeTime, err := strconv.Atoi(os.Getenv("WRITE_TIMEOUT"))
	if err != nil {
		return nil, errors.New("failed to convert WRITE_TIMEOUT to int")
	}

	idleTime, err := strconv.Atoi(os.Getenv("IDLE_TIMEOUT"))
	if err != nil {
		return nil, errors.New("failed to convert IDLE_TIMEOUT to int")
	}

	port := os.Getenv("PORT")
	if port == "" {
		return nil, errors.New("environment variable PORT is not set")
	}

	srv := serverConfig{
		Port:         port,
		ReadTimeOut:  time.Second * time.Duration(readTime),
		WriteTimeOut: time.Second * time.Duration(writeTime),
		IdleTimeout:  time.Second * time.Duration(idleTime),
	}

	setConfig.Server = srv

	setConfig.DBConnectionString = os.Getenv("DB_CONNECTION_STRING")
	if setConfig.DBConnectionString == "" {
		return nil, errors.New("environment variable DB_CONNECTION_STRING is not set")
	}

	// get the configs for the services
	FitbitConfig, err := addPlatformConfig("FITBIT")
	if err != nil {
		return nil, err
	}

	setConfig.Fitbit = FitbitConfig

	GoogleConfig, err := addPlatformConfig("GOOGLE")
	if err != nil {
		return nil, err
	}

	setConfig.Google = GoogleConfig

	StravaConfig, err := addPlatformConfig("STRAVA")
	if err != nil {
		return nil, err
	}

	setConfig.Strava = StravaConfig

	return &setConfig, nil
}

func addPlatformConfig(service string) (platformConfig, error) {
	// Create the platformConfig we will return back
	newService := platformConfig{}

	secretKey := "CLIENT_SECRET_" + service
	clientIDKey := "CLIENT_ID_" + service

	// Start parsing the config variables
	clientID := os.Getenv(clientIDKey)
	if clientID == "" {
		return newService, errors.New("environment variable [" + clientIDKey + "] does not exist")
	}

	clientSecret := os.Getenv("CLIENT_SECRET_" + service)
	if clientSecret == "" {
		return newService, errors.New("environment variable [" + secretKey + "] does not exist")
	}

	// We got they keys, so we're fine
	newService.ClientSecret = clientSecret
	newService.ClientID = clientID

	return newService, nil
}
