package environment

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// the overall structure that will contain our environment configs for the marathon service
type MarathonConfig struct {
	Server             serverConfig
	DBConnectionString string
	Fitbit             platformConfig
	Callback           string        // this will be the callback for all services. If we need multiple, this may need to change
	ClientTimeout      time.Duration // the timeout for the client that is used to make requests for Marathon
	MarathonWebsiteURL string        // we will only accept client signup requests if ti comes from the Marathon website
}

// server config options
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

// InitializeEnvironmentConfig takes the environment variables, and puts them all into an EnvironmentConfig struct
func ReadEnvFile(env string) (*MarathonConfig, error) {
	// create the Environment Config struct we will return to the user
	setConfig := MarathonConfig{}

	if env == "development" {
		// Set environment vars using .env file
		err := godotenv.Load()
		if err != nil {
			return nil, err
		}
	}

	// get the callback for all services
	callbackUrl, keyExists := os.LookupEnv("CALLBACK")
	if !keyExists {
		return nil, errors.New("environment variable [CALLBACK] does not exist")
	} else {
		setConfig.Callback = callbackUrl
	}

	// get the Marathon url
	marathonURL, keyExists := os.LookupEnv("MARATHON_WEBSITE_URL")
	if !keyExists {
		return nil, errors.New("environment variable [MARATHON_WEBSITE_URL] does not exist")
	} else {
		setConfig.MarathonWebsiteURL = marathonURL
	}

	// get the client timeout
	clientTimeout, err := strconv.Atoi(os.Getenv("CLIENT_TIMEOUT"))
	if err != nil {
		return nil, errors.New("environment variable [CLIENT_TIMEOUT] does not exist")
	} else {
		setConfig.ClientTimeout = time.Second * time.Duration(clientTimeout)
	}

	// start parsing the environment variables
	readTime, err := strconv.Atoi(os.Getenv("READ_TIMEOUT"))
	if err != nil {
		return nil, err
	}

	writeTime, err := strconv.Atoi(os.Getenv("WRITE_TIMEOUT"))
	if err != nil {
		return nil, err
	}

	idleTime, err := strconv.Atoi(os.Getenv("IDLE_TIMEOUT"))
	if err != nil {
		return nil, err
	}

	srv := serverConfig{
		Port:         os.Getenv("PORT"),
		ReadTimeOut:  time.Second * time.Duration(readTime),
		WriteTimeOut: time.Second * time.Duration(writeTime),
		IdleTimeout:  time.Second * time.Duration(idleTime),
	}

	setConfig.Server = srv

	setConfig.DBConnectionString = os.Getenv("DB_CONNECTION_STRING")

	// get the configs for the services
	FitBitConfig, err := addPlatformConfig("FITBIT")

	if err != nil {
		return nil, err
	} else {
		setConfig.Fitbit = FitBitConfig
	}

	return &setConfig, nil
}

func addPlatformConfig(service string) (platformConfig, error) {
	// create the platformConfig we will return back
	newService := platformConfig{}

	secretKey := "CLIENT_SECRET_" + service
	clientIDKey := "CLIENT_ID_" + service

	// start parsing the  config variables
	ClientID, KeyExists := os.LookupEnv(clientIDKey)
	if !KeyExists {
		return newService, errors.New("environment variable [" + clientIDKey + "] does not exist")
	}

	ClientSecret, KeyExists := os.LookupEnv("CLIENT_SECRET_" + service)
	if !KeyExists {
		return newService, errors.New("environment variable [" + secretKey + "] does not exist")
	}

	// we got they keys, so we're fine
	newService.ClientSecret = ClientSecret
	newService.ClientID = ClientID

	return newService, nil
}
