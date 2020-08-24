package writer

import (
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"

	"ygo-card-processor/models"
)

func WriteToXlsx(list models.CompleteCardList) error {
	wb := xlsx.NewFile()
	sh, err := wb.AddSheet("Card Prices")
	if err != nil {
		logrus.WithError(err).Error("Error creating sheet")
		return err
	}

	for i := 0; i < len(list.Names)+2; i++ {
		sh.AddRow()
		for j := 0; j < 4; j++ {
			sh.Rows[i].AddCell()
		}
	}

	sh.Rows[0].Cells[0].Value = "Name"
	sh.Rows[0].Cells[1].Value = "Serial"
	sh.Rows[0].Cells[2].Value = "Market"
	sh.Rows[0].Cells[3].Value = "Minimum"

	sh.Rows[len(sh.Rows)-1].Cells[2].Value = "0"
	sh.Rows[len(sh.Rows)-1].Cells[3].Value = "0"

	var marketSum float64
	var newMarketPrice float64
	var minSum float64
	var newMinPrice float64

	var row *xlsx.Row
	for i := 1; i < len(list.Names)+1; i++ {
		row = sh.Rows[i]

		row.Cells[0].Value = list.Names[i-1]
		row.Cells[1].Value = list.Serials[i-1]
		row.Cells[2].Value = list.MarketPrice[i-1]
		row.Cells[3].Value = list.AsLowAsPrice[i-1]

		if !strings.Contains(row.Cells[2].Value, "Unable") {
			marketSum, err = strconv.ParseFloat(sh.Rows[len(sh.Rows)-1].Cells[2].Value, 32)
			if err != nil {
				logrus.WithError(err).Error("Error parsing string to float")
			}
			newMarketPrice, err = strconv.ParseFloat(row.Cells[2].Value[1:], 32)
			if err != nil {
				logrus.WithError(err).Error("Error parsing string to float")
			}
		}
		if !strings.Contains(row.Cells[3].Value, "Unable") {
			minSum, err = strconv.ParseFloat(sh.Rows[len(sh.Rows)-1].Cells[3].Value, 32)
			if err != nil {
				logrus.WithError(err).Error("Error parsing string to float")
			}
			newMinPrice, err = strconv.ParseFloat(row.Cells[3].Value[1:], 32)
			if err != nil {
				logrus.WithError(err).Error("Error parsing string to float")
			}
		}

		sh.Rows[len(sh.Rows)-1].Cells[2].Value = strconv.FormatFloat(newMarketPrice+marketSum, 'f', 2, 32)
		sh.Rows[len(sh.Rows)-1].Cells[3].Value = strconv.FormatFloat(newMinPrice+minSum, 'f', 2, 32)
	}
	sh.Rows[len(sh.Rows)-1].Cells[2].Value = "$" + sh.Rows[len(sh.Rows)-1].Cells[2].Value
	sh.Rows[len(sh.Rows)-1].Cells[3].Value = "$" + sh.Rows[len(sh.Rows)-1].Cells[3].Value

	sh.Rows[len(sh.Rows)-1].Cells[0].Value = "Totals"

	if err := wb.Save("./output.xlsx"); err != nil {
		return err
	}

	return nil
}
