package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"ygo-card-processor/models"
	"ygo-card-processor/pkg/scraper"
)

func Process(cardList []string) []models.Card {
	ccl := []models.Card{}

	marketPriceReg := regexp.MustCompile(`<dd>\$[0-9]+\.[0-9]{2}`)
	asLowAsReg := regexp.MustCompile(`lowest-price-value">\$[0-9]+\.[0-9]{2}`)
	nameReg := regexp.MustCompile(`<a class="product__name".+<\/a>`)
	errorReg := regexp.MustCompile(`Oh no! Nothing was found!`)

	processedCards := make(map[string]bool)
	var invalidCards []string

	for i := range cardList {
		if processedCards[cardList[i]] {
			continue
		}
		processedCards[cardList[i]] = true

		logrus.WithFields(logrus.Fields{
			"card_serial": cardList[i],
			"progress":    fmt.Sprintf("%v of %v cards", i, len(cardList)),
		}).Info("Retrieving card info")

		htmlBytes, responseCode, err := scraper.Scrape(fmt.Sprintf("https://www.tcgplayer.com/search/yugioh/product?Number=MVP1-ENSV4", cardList[i]))
		if err != nil {
			logrus.WithError(err).Error("Error pinging url")
			continue
		} else if responseCode != 200 {
			logrus.WithField("response_code", responseCode).Info("Non-200 response code recieved from URL")
			continue
		} else if errorReg.Find(htmlBytes) != nil {
			logrus.WithField("card_serial", cardList[i]).Error("No card found with given serial")
			invalidCards = append(invalidCards, cardList[i])
			continue
		}

		marketPrice := marketPriceReg.Find(htmlBytes)
		marketPriceString := string(marketPrice)
		if marketPriceString != "" {
			index := strings.Index(marketPriceString, "$")
			marketPriceString = marketPriceString[index:]
		} else {
			marketPriceString = "Unable to get market price"
		}

		logrus.Info(string(htmlBytes))
		lowestPrice := asLowAsReg.Find(htmlBytes)
		lowestPriceString := string(lowestPrice)
		if lowestPriceString != "" {
			index := strings.Index(lowestPriceString, "$")
			lowestPriceString = lowestPriceString[index:]
		} else {
			lowestPriceString = "Unable to get lowest price"
		}

		nameRegString := string(nameReg.Find(htmlBytes))
		index := strings.Index(nameRegString, `);">`)
		logrus.Info(nameRegString)
		nameRegString = nameRegString[index+4 : len(nameRegString)-4]

		ccl = append(ccl, models.Card{
			nameRegString,
			cardList[i],
			marketPriceString,
			lowestPriceString,
		})
	}

	if len(invalidCards) > 0 {
		for j := range invalidCards {
			logrus.WithField("card_serial", invalidCards[j]).Warn("Unable to retrieve card info")
		}
	}

	return ccl
}
