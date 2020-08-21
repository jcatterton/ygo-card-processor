package processor

import (
	"ygo-card-processor/models"
	"ygo-card-processor/pkg/scraper"
	"fmt"
	"github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

func Process(cardList []string) models.CompleteCardList {
	ccl := models.CompleteCardList{
		Names: 			[]string{},
		Serials:		[]string{},
		LowestPrices:	[]string{},
		HighestPrices:	[]string{},
		AveragePrices:	[]string{},
	}

	re := regexp.MustCompile(`\$[0-9]+.[0-9]{2}`)
	errorReg := regexp.MustCompile(`No cards matching this name were found.`)
	nameReg := regexp.MustCompile(`<h1 id='item_name'>\n.+\n.+`)

	for i := range cardList {
		logrus.WithField("card_serial", cardList[i]).Info("Retrieving card info")
		htmlBytes, responseCode, err := scraper.Scrape(fmt.Sprintf("http://www.yugiohprices.com/price_history/%v", strings.ToUpper(cardList[i])))
		if err != nil {
			logrus.WithError(err).Error("Error pinging url")
		} else if responseCode != 200 {
			logrus.WithField("response_code", responseCode).Info("Non-200 response code recieved from URL")
		}

		if errorReg.Find(htmlBytes) != nil {
			logrus.WithField("card_serial", cardList[i]).Error("Unable to find information on card")
			continue
		}

		prices := re.FindAll(htmlBytes, 3)
		stringPrices := make([]string, len(prices))
		nameRegString := string(nameReg.Find(htmlBytes))

		for i := 0; i < 3; i++ {
			index := strings.Index(nameRegString, "\n")
			nameRegString = nameRegString[index + 1:]
		}

		cardName := nameRegString

		for j := range prices {
			stringPrices[j] = string(prices[j])
		}

		ccl.Names = append(ccl.Names, cardName)
		ccl.Serials = append(ccl.Serials, cardList[i])
		ccl.LowestPrices = append(ccl.LowestPrices, stringPrices[0])
		ccl.HighestPrices = append(ccl.HighestPrices, stringPrices[1])
		ccl.AveragePrices = append(ccl.AveragePrices, stringPrices[2])
	}

	return ccl
}
