package server

import (
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/swishcloud/goweb"
)

const (
	API_PATH_File = "/api/file"
)

func (s *FileSyncWebServer) bindApiHandlers(root goweb.RouterGroup) {
	root.POST(API_PATH_File, s.fileApiPostHandler())
}

func (s *FileSyncWebServer) fileApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		file, fileHeader, err := ctx.Request.FormFile("file")
		if err != nil {
			panic(err)
		}
		uuid := uuid.New().String() + filepath.Ext(fileHeader.Filename)
		path := s.config.upload_folder + uuid
		out, err := os.Create(path)
		defer out.Close()
		if err != nil {
			panic(err)
		}
		io.Copy(out, file)
		ctx.Writer.WriteHeader(204)
	}
}
