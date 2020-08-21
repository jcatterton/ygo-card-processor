package scraper

import (
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

func Scrape(u string) ([]byte, int, error) {
	client := http.Client{}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		logrus.WithError(err).Error("Error while creating request")
		return nil, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error while executing request")
		return nil, 0, err
	}

	decodedBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.WithError(err).Error("Error while reading response body")
	}

	if err := resp.Body.Close(); err != nil {
		logrus.WithError(err).Error("Error closing request body")
	}

	return decodedBody, resp.StatusCode, nil
}
