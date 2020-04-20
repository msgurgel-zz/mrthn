package platform

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"github.com/msgurgel/marathon/pkg/auth"
	"github.com/msgurgel/marathon/pkg/dal"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Google struct {
	db            *sql.DB
	log           *logrus.Logger
	domain        string
	authorization *oauth2.Config
}

// Appended to the end of every call for Google fit for aggregated data
const googleFitEndpoint string = "/users/me/dataset:aggregate"

// The following are data source IDs for the Google Fit datasources we are querying
const aggregatedStepsID string = "derived:com.google.step_count.delta:com.google.android.gms:estimated_steps"
const aggregatedCaloriesID string = "derived:com.google.calories.expended:com.google.android.gms:merge_calories_expended"
const aggregatedDistanceID string = "derived:com.google.distance.delta:com.google.android.gms:merge_distance_delta"

const millisecondsInADay int64 = 86400000

// GoogleFitRequest is a struct for sending a Google request
type GoogleFitRequest struct {
	AggregateBy     []map[string]string `json:"aggregateBy"`
	BucketByTime    map[string]int64    `json:"bucketByTime"`
	StartTimeMillis int64               `json:"startTimeMillis"`
	EndTimeMillis   int64               `json:"endTimeMillis"`
}

// periodToMilliseconds maps a valid period string to their corresponding milliseconds
var periodToMilliseconds = map[string]int64{
	"1d":  86400000,
	"7d":  608400000,
	"30d": 2592000000,
	"1w":  608400000,
	"1m":  2592000000,
	"3m":  7776000000,
	"6m":  23330000000,
}

// GoogleValueResponse is the struct that contains the value of the datapoint we requested from Google Fit
type GoogleValuesResponse = []map[string]interface{}

type DataSet struct {
	DataSourceID string  `json:"dataSourceId"`
	Points       []Point `json:"point"`
}

type Point struct {
	Values GoogleValuesResponse `json:"value,omitempty"`
}

type Bucket struct {
	Datasets []DataSet `json:"dataset"`
}

type Error struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type GoogleFitWholeResponse struct {
	Buckets []Bucket `json:"bucket"`
	Error   Error    `json:"error,omitempty"`
}

func (g Google) Name() string {
	return "google"
}

func (g Google) GetSteps(userID int, date time.Time) (int, error) {
	response, err := g.makeGoogleFitRequest(userID, date, aggregatedStepsID, "intVal", "")
	if err != nil {
		return 0, err
	}

	// Google gives the value back as a float, but it can be parsed as an int

	return int(response), nil
}

func (g Google) GetCalories(userID int, date time.Time) (int, error) {
	response, err := g.makeGoogleFitRequest(userID, date, aggregatedCaloriesID, "fpVal", "")

	if err != nil {
		return 0, err
	}

	return int(response), nil
}

func (g Google) GetDistance(userID int, date time.Time) (float64, error) {
	response, err := g.makeGoogleFitRequest(userID, date, aggregatedDistanceID, "fpVal", "")
	if err != nil {
		return 0, err
	}

	// To prevent dividing by 0
	if response == 0 {
		return 0, nil
	}

	// Divide the result by 1000, because Google Fit returns meters when we want km
	return response / 1000, nil
}

func (g Google) GetDistanceOverPeriod(userID int, date time.Time, period string) (float64, error) {
	response, err := g.makeGoogleFitRequest(userID, date, aggregatedDistanceID, "fpVal", period)
	if err != nil {
		return 0, err
	}

	// To prevent dividing by 0
	if response == 0 {
		return 0, nil
	}

	// Divide the result by 1000, because Google Fit returns meters when we want km
	return response / 1000, nil
}

func (g Google) makeGoogleFitRequest(userID int, date time.Time, dataSourceID string, valueName string, period string) (float64, error) {
	// Get Access Token associated with user from db
	tokens, err := dal.GetUserTokens(g.db, userID, g.Name())

	// Before we can make the request, refresh the access tokens
	newTokens, err := auth.RefreshOAuth2Tokens(tokens, g.authorization)
	if err != nil {
		return 0, err
	}

	if newTokens.AccessToken != tokens.AccessToken {
		// Tokens were updated, let's update the database
		err := dal.UpdateCredentialsUsingOAuth2Tokens(g.db, userID, newTokens)
		if err != nil {
			return 0, errors.New("failed to update db with new oauth2 tokens: " + err.Error())
		}

		g.log.WithFields(logrus.Fields{
			"user":   userID,
			"expiry": newTokens.Expiry,
		}).Info("updated access token")
	}

	// Tokens were refreshed. Prepare to make the request.
	client := g.authorization.Client(context.Background(), newTokens)

	url := g.domain + googleFitEndpoint

	UnixTimeDateInt := date.UnixNano() / 1000000

	var bucketByTime = make(map[string]int64)

	var UnixTimeLimit int64
	// If the user only entered the date, just add the amount of milliseconds in a day to the time limit
	if period == "" {
		UnixTimeLimit = UnixTimeDateInt + millisecondsInADay
		bucketByTime["durationMillis"] = millisecondsInADay
	} else {
		// Get the appropriate amount of milliseconds from the map
		if millisecondValue, ok := periodToMilliseconds[period]; ok {
			UnixTimeLimit = UnixTimeDateInt + millisecondValue

			bucketByTime["durationMillis"] = UnixTimeLimit - UnixTimeDateInt
		} else {
			// Something went wrong, we are only supposed to get valid periods from earlier in the call layer
			g.log.WithFields(logrus.Fields{
				"error":    "period value received was not valid",
				"received": period,
				"function": "makeGoogleFitRequest",
			}).Error("Improper period passed in")

			return 0, errors.New("invalid period value")
		}
	}

	aggregateBy := make([]map[string]string, 1)
	dataSourceMap := make(map[string]string)
	dataSourceMap["dataSourceId"] = dataSourceID
	aggregateBy[0] = dataSourceMap

	// Need to create the request body
	requestBody, err := json.Marshal(GoogleFitRequest{
		AggregateBy:     aggregateBy,
		BucketByTime:    bucketByTime,
		StartTimeMillis: UnixTimeDateInt,
		EndTimeMillis:   UnixTimeLimit,
	})

	if err != nil {
		g.log.WithFields(logrus.Fields{
			"error":           err.Error(),
			"aggregateBy":     aggregateBy,
			"bucketByTime":    bucketByTime,
			"startTimeMillis": UnixTimeDateInt,
			"endTimeMillis":   UnixTimeLimit,
		}).Error("failed to marshal google fit request")

		return 0, err
	}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		g.log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to request data from Google Fit")

		return 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		g.log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("failed to read response data from Google Fit")

		return 0, nil
	}
	_ = resp.Body.Close()

	// Unmarshal the JSON response into a Google Fit Response struct
	responseValue := GoogleFitWholeResponse{}
	err = json.Unmarshal(body, &responseValue)
	if err != nil {
		g.log.WithFields(logrus.Fields{
			"error":        err.Error(),
			"responseBody": string(body),
		}).Error("failed to unmarshal Google Fit response")

		return 0, err
	}

	// First, check if there was an error in the response
	if responseValue.Error.Message != "" {
		g.log.WithFields(logrus.Fields{
			"error":        responseValue.Error.Message,
			"code":         responseValue.Error.Code,
			"responseBody": string(body),
		}).Error("received bad response from Google Fit")
		return 0, nil
	}

	result := 0.0

	for _, currentBucket := range responseValue.Buckets {

		if len(currentBucket.Datasets) < 1 {
			continue
		}

		for _, currentDataSet := range currentBucket.Datasets {

			// if the currentData set is empty, continue
			if len(currentDataSet.Points) < 1 {
				continue
			}

			for _, currentPoint := range currentDataSet.Points {
				// Add the result to the total
				result += currentPoint.Values[0][valueName].(float64)
			}
		}
	}
	return result, nil
}
