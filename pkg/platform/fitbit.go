package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/msgurgel/mrthn/pkg/auth"

	"golang.org/x/oauth2"

	"github.com/msgurgel/mrthn/pkg/helpers"

	"github.com/sirupsen/logrus"

	"github.com/msgurgel/mrthn/pkg/dal"
)

// ResourceEndpoint contains endpoint for any type of resource we want to access from Fitbit
var resourceEndpoints = map[int]string{
	stepType:     "activities/steps",
	distanceType: "activities/distance",
	caloriesType: "activities/calories",
}

type Fitbit struct {
	db            *sql.DB
	log           *logrus.Logger
	domain        string
	authorization *oauth2.Config
}

type Summary struct {
	Calories int                      `json:"caloriesOut"`
	Steps    int                      `json:"steps"`
	Distance []map[string]interface{} `json:"distances"`
}

type dailyActivity struct {
	Summary Summary             `json:"summary"`
	Errors  []map[string]string `json:"errors,omitempty"`
}

type dailyResourceSummary struct {
	DateTime string `json:"dateTime"`
	Value    string `json:"value"`
}

const stepType = 1
const distanceType = 2
const caloriesType = 3

var resourceNames = map[int]string{
	stepType:     "activities-steps",
	distanceType: "activities-distance",
	caloriesType: "activities-calories",
}

func (f Fitbit) Name() string {
	return "fitbit"
}

func (f Fitbit) GetSteps(user int, date time.Time) (int, error) {
	dailyAct, err := f.getDailyActivity(user, date)
	if err != nil {
		return 0, err
	}

	return dailyAct.Summary.Steps, nil
}

func (f Fitbit) GetCalories(user int, date time.Time) (int, error) {
	dailyAct, err := f.getDailyActivity(user, date)
	if err != nil {
		return 0, err
	}

	return dailyAct.Summary.Calories, nil
}

func (f Fitbit) GetDistance(user int, date time.Time) (float64, error) {
	dailyAct, err := f.getDailyActivity(user, date)
	if err != nil {
		return 0, err
	}

	// We just want the total distance of all the activities returned. The total distance is in the first map
	return dailyAct.Summary.Distance[0]["distance"].(float64), nil
}

func (f Fitbit) GetDistanceOverPeriod(user int, date time.Time, period string) (float64, error) {
	tokens, err := dal.GetUserTokens(f.db, user, f.Name())
	if err != nil {
		return 0, err
	}

	result, err := f.callActivityTimeSeries(user, tokens, distanceType, date, period)

	if err != nil {
		return 0, nil
	}

	return result, nil
}

func (f *Fitbit) callActivityTimeSeries(userID int, tokens *oauth2.Token, resourceType int, date time.Time, period string) (float64, error) {
	// Get Access Token associated with user from db
	newTokens, err := auth.RefreshOAuth2Tokens(tokens, f.authorization)
	if err != nil {
		return 0, err
	}

	if newTokens.AccessToken != tokens.AccessToken {
		// Tokens were updated, let's update the database
		err := dal.UpdateCredentialsUsingOAuth2Tokens(f.db, userID, newTokens)
		if err != nil {
			return 0, errors.New("failed to update db with new oauth2 tokens: " + err.Error())
		}

		f.log.WithFields(logrus.Fields{
			"user":   userID,
			"expiry": newTokens.Expiry,
		}).Info("updated access token")
	}

	// Form the activity series URL
	url := fmt.Sprintf("%s/user/-/%s/date/%s/%s.json", f.domain, resourceEndpoints[resourceType], date.Format(helpers.ISOLayout), period)

	// Tokens were refreshed. Now make the request
	client := f.authorization.Client(context.Background(), newTokens)
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil
	}
	_ = resp.Body.Close()

	// Depending on what type of resource we requested, the Activities can come within one of three types of structures

	// First, attempt to marshal the request body into a list of dailyResourceSummary
	var data map[string][]dailyResourceSummary
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}

	totalValue := 0.0

	for _, currentActivity := range data[resourceNames[resourceType]] {
		if currentValue, err := strconv.ParseFloat(currentActivity.Value, 64); err == nil {
			totalValue += currentValue
		} else {
			// Log the error and continue
			f.log.WithFields(logrus.Fields{
				"value": currentActivity.Value,
				"error": "attempt to parse value from Fitbit failed",
			}).Error("bad value received from Fitbit")
			continue
		}
	}

	return totalValue, nil
}

func (f Fitbit) getDailyActivity(userID int, date time.Time) (dailyActivity, error) {
	// Get Access Token associated with user from db
	tokens, err := dal.GetUserTokens(f.db, userID, f.Name())
	if err != nil {
		return dailyActivity{}, err
	}

	// Call fitbit endpoint passing access token and date
	dailyAct, err := f.callDailyActivityEndpoint(
		f.domain+"/user/-/activities/date",
		userID,
		tokens,
		date,
	)
	if err != nil {
		return dailyActivity{}, err
	}
	return dailyAct, nil
}

func (f *Fitbit) callDailyActivityEndpoint(url string, userID int, tokens *oauth2.Token, date time.Time) (dailyActivity, error) {
	// Add date to end of the Daily Activity URL
	url = fmt.Sprintf("%s/%s.json", url, date.Format(helpers.ISOLayout))
	newTokens, err := auth.RefreshOAuth2Tokens(tokens, f.authorization)
	if err != nil {
		return dailyActivity{}, err
	}

	if newTokens.AccessToken != tokens.AccessToken {
		// Tokens were updated, let's update the database
		err := dal.UpdateCredentialsUsingOAuth2Tokens(f.db, userID, newTokens)
		if err != nil {
			return dailyActivity{}, errors.New("failed to update db with new oauth2 tokens: " + err.Error())
		}

		f.log.WithFields(logrus.Fields{
			"user":   userID,
			"expiry": newTokens.Expiry,
		}).Info("updated access token")
	}

	// Tokens were refreshed. Now make the request
	client := f.authorization.Client(context.Background(), newTokens)
	resp, err := client.Get(url)
	if err != nil {
		return dailyActivity{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return dailyActivity{}, nil
	}
	_ = resp.Body.Close()

	// Unmarshal the JSON response into a Daily Activity struct
	dailyAct := dailyActivity{}
	err = json.Unmarshal(body, &dailyAct)
	if err != nil {
		return dailyActivity{}, err
	}

	if len(dailyAct.Errors) > 0 {
		for i, e := range dailyAct.Errors {
			f.log.WithFields(logrus.Fields{
				"errorType": e["errorType"],
				"message":   e["message"],
			}).Errorf("request to fitbit api failed - reason %d", i+1)
		}

		return dailyActivity{}, errors.New("failed to request daily activity")
	}

	return dailyAct, nil
}
