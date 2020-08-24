package main

import (
	"github.com/sirupsen/logrus"
	"ygo-card-processor/pkg/processor"
	"ygo-card-processor/pkg/reader"
	"ygo-card-processor/pkg/writer"
)

func main() {
	cardList, err := reader.OpenAndReadFile("cardlist.xlsx")
	if err != nil {
		logrus.WithError(err).Error("Error opening/reading file")
	}

	ccl := processor.Process(cardList)
	logrus.WithField("completed_card_list_length", len(ccl.Names)).Info("Retrieved price info on cards")

	if err := writer.WriteToXlsx(ccl); err != nil {
		logrus.WithError(err).Error("Error while writing to excel file")
	}
}
