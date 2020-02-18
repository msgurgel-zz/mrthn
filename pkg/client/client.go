package client

import (
	"io/ioutil"
	"net/http"
	"time"
)

func NewClient(timeoutInSec int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeoutInSec) * time.Second,
	}
}

func SetOAuth2ReqHeaders(r *http.Request, accessToken string) *http.Request {
	r.Header.Set("User-Agent", "marathon")
	r.Header.Set("Content-type", "application/json")
	r.Header.Set("Authorization", "Bearer "+accessToken)

	return r
}

func MakeRequest(client *http.Client, r *http.Request) ([]byte, int, error) {
	res, err := client.Do(r)
	if err != nil {
		return nil, 0, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}
	_ = res.Body.Close()

	return body, res.StatusCode, nil
}

func PrepareGETRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}
