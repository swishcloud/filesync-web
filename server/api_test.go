package server

import (
	"net/http/httptest"
	"testing"

	"github.com/swishcloud/goweb"
)

func TestPostfileHandler(t *testing.T) {
	r := testApiHandler("POST", API_PATH_File, func(s *FileSyncWebServer) goweb.HandlerFunc { return s.fileApiPostHandler() })
	if r.Code != 204 {
		t.FailNow()
	}
}
func testApiHandler(method, path string, getHandler func(*FileSyncWebServer) goweb.HandlerFunc) *httptest.ResponseRecorder {
	server := NewFileSyncWebServer("config.yaml", false)
	server.engine.POST(path, getHandler(server))
	ts := httptest.NewTLSServer(server.engine)
	req := httptest.NewRequest(method, ts.URL+path, nil)
	w := httptest.NewRecorder()
	server.engine.ServeHTTP(w, req)
	return w
}
