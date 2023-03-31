package internal

import (
	"regexp"
	"strings"
)

type FILE_TYPE = int

const (
	FILE      FILE_TYPE = 1
	Directory FILE_TYPE = 2
)

func ContentTypeFromExpansion(expansion string) string {
	expansion = strings.ToLower(expansion)
	switch expansion {
	case ".jpeg":
		return "image/jpeg"
	case ".sh":
		fallthrough
	case ".txt":
		return ""
	case ".csv":
		return "text/csv"
	case ".html":
		return "text/html"
	}
	return "application/octet-stream"
}
func ExpansionFromFileName(filename string) string {
	reg, err := regexp.Compile(`\.[\d\w]+$`)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(reg.FindString(filename))
}
