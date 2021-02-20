package reader

import (
	"bytes"
	"io"
	"mime/multipart"

	"github.com/tealeg/xlsx"
)

func OpenAndReadFile(file multipart.File) ([]string, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}

	wb, err := xlsx.OpenBinary(buf.Bytes())
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
