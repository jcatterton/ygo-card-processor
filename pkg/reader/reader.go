package reader

import "mime/multipart"

type FileReader interface {
	OpenAndReadFile(file multipart.File) ([]string, error)
}
