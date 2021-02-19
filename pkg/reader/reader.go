package reader

import (
	"fmt"
	"ygo-card-processor/models"

	"github.com/tealeg/xlsx"
)

func OpenAndReadFile(path string) ([]models.Card, error) {
	wb, err := xlsx.OpenFile(fmt.Sprintf("%v", path))
	if err != nil {
		return nil, err
	}

	var cardList []models.Card

	sh := wb.Sheets[0]
	for i := 0; i < len(sh.Rows); i++ {
		cardList = append(cardList, models.Card{
			Name:        sh.Rows[i].Cells[0].Value,
			Serial:      sh.Rows[i].Cells[1].Value,
			MarketPrice: sh.Rows[i].Cells[2].Value,
			LowestPrice: sh.Rows[i].Cells[3].Value,
		})
	}

	return cardList, nil
}
