package writer

import (
	"github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx"
	"strconv"
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
		for j := 0; j < 5; j++ {
			sh.Rows[i].AddCell()
		}
	}

	sh.Rows[0].Cells[0].Value = "Name"
	sh.Rows[0].Cells[1].Value = "Serial"
	sh.Rows[0].Cells[2].Value = "Minimum"
	sh.Rows[0].Cells[3].Value = "Maximum"
	sh.Rows[0].Cells[4].Value = "Average"

	var row *xlsx.Row
	for i := 1; i < len(list.Names)+1; i++ {
		row = sh.Rows[i]

		row.Cells[0].Value = list.Names[i-1]
		row.Cells[1].Value = list.Serials[i-1]
		row.Cells[2].Value = list.LowestPrices[i-1]
		row.Cells[3].Value = list.HighestPrices[i-1]
		row.Cells[4].Value = list.AveragePrices[i-1]

		min, _ := strconv.ParseFloat(sh.Rows[len(sh.Rows)-1].Cells[2].Value, 32)
		minSum, _ := strconv.ParseFloat(row.Cells[2].Value[1:], 32)
		max, _ := strconv.ParseFloat(sh.Rows[len(sh.Rows)-1].Cells[3].Value, 32)
		maxSum, _ := strconv.ParseFloat(row.Cells[3].Value[1:], 32)
		avg, _ := strconv.ParseFloat(sh.Rows[len(sh.Rows)-1].Cells[4].Value, 32)
		avgSum, _ := strconv.ParseFloat(row.Cells[4].Value[1:], 32)

		sh.Rows[len(sh.Rows)-1].Cells[2].Value = strconv.FormatFloat(minSum+min, 'f', 2, 32)
		sh.Rows[len(sh.Rows)-1].Cells[3].Value = strconv.FormatFloat(maxSum+max, 'f', 2, 32)
		sh.Rows[len(sh.Rows)-1].Cells[4].Value = strconv.FormatFloat(avgSum+avg, 'f', 2, 32)
	}
	sh.Rows[len(sh.Rows)-1].Cells[0].Value = "Totals"

	if err := wb.Save("./output.xlsx"); err != nil {
		return err
	}

	return nil
}
