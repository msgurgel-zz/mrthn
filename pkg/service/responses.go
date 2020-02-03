/*
 * Marathon API
 *
 * One login for all your fitness data needs.
 *
 * API version: 0.1.0
 */
package service

type GetUserStepsResponse200 struct {
	Id int `json:"id,omitempty"`

	Steps int `json:"steps,omitempty"`
}

type GetUserCaloriesResponse200 struct {
	Id int `json:"id,omitempty"`

	Calories int `json:"calories,omitempty"`
}
