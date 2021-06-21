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
	"github.com/msgurgel/mrthn/pkg/dal"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Strava struct {
	db            *sql.DB
	log           *logrus.Logger
	domain        string
	authorization *oauth2.Config
}

// The endpoint for Strava activities
const stravaActivityEndpoint string = "/athlete/activities"

const secondsInADay int64 = 86400

// StravaActivity represents an activity that would be returned by the query
type StravaActivity struct {
	Distance   float64 `json:"distance,omitempty"`
	Kilojoules float64 `json:"kilojoules,omitempty"`
}

// ActivityStats represents the aggregated activity data of a returned query
type ActivityStats struct {
	totalCalories int
	totalDistance float64
}

// periodToSeconds maps a valid period string to their corresponding seconds value
var periodToSeconds = map[string]int64{
	"1d":  86400,
	"7d":  608400,
	"30d": 2592000,
	"1w":  608400,
	"1m":  2592000,
	"3m":  7776000,
	"6m":  23330000,
}

func (s Strava) Name() string {
	return "strava"
}

func (s Strava) GetSteps(userID int, date time.Time) (int, error) {
	return 0, nil
}

func (s Strava) GetCalories(userID int, date time.Time) (int, error) {
	dailyAct, err := s.getStravaActivityCount(userID, date, "")
	if err != nil {
		return 0, err
	}

	return dailyAct.totalCalories, nil
}

func (s Strava) GetDistance(userID int, date time.Time) (float64, error) {

	dailyAct, err := s.getStravaActivityCount(userID, date, "")
	if err != nil {
		return 0, err
	}

	// mrthn returns distances in kilometres, not meters
	kilometerValue := dailyAct.totalDistance / 1000

	return kilometerValue, nil
}

func (s Strava) GetDistanceOverPeriod(userID int, date time.Time, period string) (float64, error) {
	activityStats, err := s.getStravaActivityCount(userID, date, period)
	if err != nil {
		return 0, err
	}

	// mrthn returns distances in kilometers, not meters
	kilometerValue := activityStats.totalDistance / 1000

	return kilometerValue, nil
}

func (s Strava) getStravaActivityCount(userID int, date time.Time, period string) (ActivityStats, error) {
	// Get Access Token associated with user from db
	tokens, err := dal.GetUserTokens(s.db, userID, s.Name())

	// Before we can make the request, refresh the access tokens
	newTokens, err := auth.RefreshOAuth2Tokens(tokens, s.authorization)
	if err != nil {
		return ActivityStats{}, err
	}

	if newTokens.AccessToken != tokens.AccessToken {
		// Tokens were updated, let's update the database
		err := dal.UpdateCredentialsUsingOAuth2Tokens(s.db, userID, newTokens)
		if err != nil {
			return ActivityStats{}, errors.New("failed to update db with new oauth2 tokens: " + err.Error())
		}

		s.log.WithFields(logrus.Fields{
			"user":   userID,
			"expiry": newTokens.Expiry,
		}).Info("updated access token")
	}

	// Tokens were refreshed. Prepare to make the request.
	client := s.authorization.Client(context.Background(), newTokens)

	url := s.domain + stravaActivityEndpoint

	// To filter the activates received by a certain day, we need the epoch time
	epochTimeDateInt := date.Unix()
	var epochTimeLimit int64

	// If the user only entered the date, just add the amount of milliseconds in a day to the time limit
	if period == "" {
		epochTimeLimit = epochTimeDateInt + secondsInADay
	} else {
		// Get the appropriate amount of seconds from the map
		if secondValue, ok := periodToSeconds[period]; ok {
			epochTimeLimit = epochTimeDateInt + secondValue
		} else {
			// Something went wrong, we are only supposed to get valid periods from earlier in the call layer
			s.log.WithFields(logrus.Fields{
				"error":    "period value received was not valid",
				"received": period,
				"function": "getStravaActivityCount",
			}).Error("Improper period passed in")

			return ActivityStats{}, errors.New("invalid period value")
		}
	}

	// Need to add the epoch timestamps as filter queries to the URL
	url += fmt.Sprintf("?before=%s&after=%s", strconv.FormatInt(epochTimeLimit, 10), strconv.FormatInt(epochTimeDateInt, 10))

	resp, err := client.Get(url)
	if err != nil {
		return ActivityStats{}, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ActivityStats{}, nil
	}
	_ = resp.Body.Close()

	// Unmarshal the JSON response into a list of Strava Activity struct
	var activityList []StravaActivity
	err = json.Unmarshal(body, &activityList)
	if err != nil {
		return ActivityStats{}, err
	}

	// Now that we have the list of activities, scan through each activity and add the totals together
	result := ActivityStats{
		totalCalories: 0,
		totalDistance: 0,
	}

	for _, s := range activityList {

		// Depending on the activity type, it might not have distance or calories present
		if s.Distance != 0 {
			result.totalDistance += s.Distance
		}

		if s.Kilojoules != 0 {
			// Need to convert the kilojoules to calories

			caloriesCount := s.Kilojoules / 4.814

			result.totalCalories += int(caloriesCount)
		}
	}

	return result, nil
}
