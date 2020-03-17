package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/msgurgel/marathon/pkg/auth"

	"golang.org/x/oauth2"

	"github.com/msgurgel/marathon/pkg/helpers"

	"github.com/sirupsen/logrus"

	"github.com/msgurgel/marathon/pkg/dal"
)

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
