package platform

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/msgurgel/marathon/pkg/helpers"

	"github.com/sirupsen/logrus"

	"github.com/msgurgel/marathon/pkg/client"

	"github.com/msgurgel/marathon/pkg/dal"
)

type Summary struct {
	Calories int                      `json:"caloriesOut"`
	Steps    int                      `json:"steps"`
	Distance []map[string]interface{} `json:"distances"`
}

type dailyActivity struct {
	Summary Summary             `json:"summary"`
	Errors  []map[string]string `json:"errors,omitempty"`
}

type Fitbit struct {
	db     *sql.DB
	log    *logrus.Logger
	domain string
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
	// we just want the total distance of all the activities returned. The total distance is in the first map
	return dailyAct.Summary.Distance[0]["distance"].(float64), nil
}

// TODO: deal with refreshing access token
func (f Fitbit) getDailyActivity(user int, date time.Time) (dailyActivity, error) {
	// Get Access Token associated with user from db
	accessTkn, _, err := dal.GetUserTokens(f.db, user, f.Name())
	if err != nil {
		return dailyActivity{}, err
	}

	// Call fitbit endpoint passing access token and date
	dailyAct, err := f.callDailyActivityEndpoint(
		f.domain+"/user/-/activities/date",
		accessTkn,
		date,
	)
	if err != nil {
		return dailyActivity{}, err
	}
	return dailyAct, nil
}

func (f *Fitbit) callDailyActivityEndpoint(url string, accessToken string, date time.Time) (dailyActivity, error) {
	// Add date to end of the Daily Activity URL
	url = fmt.Sprintf("%s/%s.json", url, date.Format(helpers.ISOLayout))

	// Make request to Fitbit servers
	req, err := client.PrepareGETRequest(url)
	if err != nil {
		return dailyActivity{}, err
	}
	req = client.SetOAuth2ReqHeaders(req, accessToken)

	result, _, err := client.MakeRequest(client.NewClient(2), req)
	if err != nil {
		return dailyActivity{}, err
	}

	// Unmarshal the JSON response into a Daily Activity struct
	dailyAct := dailyActivity{}
	err = json.Unmarshal(result, &dailyAct)
	if err != nil {
		return dailyActivity{}, err
	}

	if len(dailyAct.Errors) > 0 {
		for i, e := range dailyAct.Errors {
			f.log.WithFields(logrus.Fields{
				"errorType": e["errorType"],
				"message":   e["message"],
			}).Errorf("request to fitbit api failed - reason %d", i)
		}

		return dailyActivity{}, errors.New("failed to request daily activity")
	}

	return dailyAct, nil
}
