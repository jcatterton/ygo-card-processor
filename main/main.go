package main

import (
	"github.com/sirupsen/logrus"
	"ygo-card-processor/pkg/api"
)

func main() {
	if err := api.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("Could not serve API")
	}
}
