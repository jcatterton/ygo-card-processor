package processor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"ygo-card-processor/models"
	"ygo-card-processor/pkg/scraper"
)

func Process(cardList []string) models.CompleteCardList {
	ccl := models.CompleteCardList{
		Names:        []string{},
		Serials:      []string{},
		MarketPrice:  []string{},
		AsLowAsPrice: []string{},
	}

	marketPriceReg := regexp.MustCompile(`<dd>\$[0-9]+\.[0-9]{2}`)
	asLowAsReg := regexp.MustCompile(`lowest-price-value">\$[0-9]+\.[0-9]{2}`)
	nameReg := regexp.MustCompile(`<a class="product__name".+<\/a>`)

	processedCards := make(map[string]bool)

	for i := range cardList {
		if processedCards[cardList[i]] {
			continue
		}
		processedCards[cardList[i]] = true

		logrus.WithFields(logrus.Fields{
			"card_serial": cardList[i],
			"progress":    fmt.Sprintf("%v of %v cards", i, len(cardList)),
		}).Info("Retrieving card info")

		htmlBytes, responseCode, err := scraper.Scrape(fmt.Sprintf("https://shop.tcgplayer.com/yugioh/product/show?advancedSearch=true&Number=%v", strings.ToUpper(cardList[i])))
		if err != nil {
			logrus.WithError(err).Error("Error pinging url")
		} else if responseCode != 200 {
			logrus.WithField("response_code", responseCode).Info("Non-200 response code recieved from URL")
		}

		marketPrice := marketPriceReg.Find(htmlBytes)
		marketPriceString := string(marketPrice)
		index := strings.Index(marketPriceString, "$")
		marketPriceString = marketPriceString[index:]

		lowestPrice := asLowAsReg.Find(htmlBytes)
		lowestPriceString := string(lowestPrice)
		index = strings.Index(lowestPriceString, "$")
		lowestPriceString = lowestPriceString[index:]

		stringPrices := []string{marketPriceString, lowestPriceString}

		nameRegString := string(nameReg.Find(htmlBytes))
		index = strings.Index(nameRegString, `);">`)
		nameRegString = nameRegString[index+4 : len(nameRegString)-4]
		cardName := nameRegString

		ccl.Names = append(ccl.Names, cardName)
		ccl.Serials = append(ccl.Serials, cardList[i])
		ccl.MarketPrice = append(ccl.MarketPrice, stringPrices[0])
		ccl.AsLowAsPrice = append(ccl.AsLowAsPrice, stringPrices[1])
	}

	return ccl
}
