package reader

import (
	"fmt"
	"github.com/tealeg/xlsx"
)

func OpenAndReadFile(path string) ([]string, error) {
	wb, err := xlsx.OpenFile(fmt.Sprintf("%v", path))
	if err != nil {
		return nil, err
	}

	var cardList []string

	sh := wb.Sheets[0]
	for i := 0; i < len(sh.Rows); i++ {
		cardList = append(cardList, sh.Rows[i].Cells[0].Value)
	}

	return cardList, nil
}
