package server

import "github.com/swishcloud/goweb"

const (
	API_PATH_File = "/api/file"
)

func (s *FileSyncWebServer) fileApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.Writer.WriteHeader(204)
	}
}
