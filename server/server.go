package server

import (
	"log"
	"net/http"

	"github.com/swishcloud/goweb"
)

type FileSyncWebServer struct {
	engine *goweb.Engine
}

func NewFileSyncWebServer() *FileSyncWebServer {
	s := new(FileSyncWebServer)
	s.engine = goweb.Default()
	return s
}
func (s *FileSyncWebServer) Serve() {
	s.bindHandlers(s.engine.RouterGroup)
	addr := ":2000"
	log.Println("listening on", addr)
	err := http.ListenAndServe(addr, s.engine)
	if err != nil {
		log.Fatal(err)
	}
}
