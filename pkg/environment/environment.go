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
	Server   serverConfig
	FitBit   serviceConfig
	Callback string // this will be the callback for all services. If we need multiple, this may need to change
}

// server config options
type serverConfig struct {
	Address      string
	ReadTimeOut  time.Duration
	WriteTimeOut time.Duration
	IdleTimeout  time.Duration
}

// Config struct specifically for Fitbit client ids, secrets, etc
type serviceConfig struct {
	ClientID     string
	ClientSecret string
}

// InitializeEnvironmentConfig takes the environment variables, and puts them all into an EnvironmentConfig struct
func InitializeEnvironmentConfig() (MarathonConfig, error) {
	// create the Environment Config struct we will return to the user
	setConfig := MarathonConfig{}

	// get the environment variables
	err := godotenv.Load()
	if err != nil {
		return setConfig, err
	} else {

		// get the callback for all services
		callbackUrl, KeyExists := os.LookupEnv("CALLBACK")
		if !KeyExists {
			return setConfig, errors.New("environment variable [CALLBACK] does not exist")
			return setConfig, err
		} else {
			setConfig.Callback = callbackUrl
		}

		// start parsing the environment variables
		readTime, err := strconv.Atoi(os.Getenv("READ_TIMEOUT"))
		if err != nil {
			return setConfig, err
		} // is there a better way to handle these?

		writeTime, err := strconv.Atoi(os.Getenv("WRITE_TIMEOUT"))
		if err != nil {
			return setConfig, err
		}

		idleTime, err := strconv.Atoi(os.Getenv("IDLE_TIMEOUT"))
		if err != nil {
			return setConfig, err
		}

		srv := serverConfig{
			Address:      os.Getenv("SERVER_ADDRESS"),
			ReadTimeOut:  time.Second * time.Duration(readTime),
			WriteTimeOut: time.Second * time.Duration(writeTime),
			IdleTimeout:  time.Second * time.Duration(idleTime),
		}

		setConfig.Server = srv

		// get the configs for the services

		FitBitConfig, err := AddServiceConfig("FITBIT")

		if err != nil {
			return setConfig, err
		} else {
			setConfig.FitBit = FitBitConfig
		}

		return setConfig, nil

	}
}

func AddServiceConfig(service string) (serviceConfig, error) {
	// create the serviceConfig we will return back
	newService := serviceConfig{}

	secretKey := "CLIENT_SECRET_" + service
	clientIdKey := "CLIENT_ID_" + service

	// start parsing the  config variables
	ClientID, KeyExists := os.LookupEnv(clientIdKey)
	if !KeyExists {
		return newService, errors.New("environment variable [" + clientIdKey + "] does not exist")
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
