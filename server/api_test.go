package server

import (
	"bytes"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/swishcloud/gostudy/common"
)

func TestPostfileHandler(t *testing.T) {
	rac := common.NewRestApiClient(true)
	at := NewApiTester()
	buf := bytes.NewBuffer([]byte{})
	w := multipart.NewWriter(buf)
	if writer, err := w.CreateFormFile("file", "test_file.tst"); err != nil {
		log.Fatal(err)
	} else {
		_, err := writer.Write([]byte("hello,world"))
		if err != nil {
			log.Fatal(err)
		}
	}
	w.Close()
	rar := common.NewRestApiRequest("POST", at.ts.URL+API_PATH_File_Upload, buf.Bytes())
	ct := w.FormDataContentType()
	t.Log(ct)
	rar.Request.Header.Set("Content-Type", ct)
	resp, err := rac.Do(rar)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Fatal(resp.Status)
	}
}

type apiTester struct {
	responseRecord *httptest.ResponseRecorder
	ts             *httptest.Server
	server         *FileSyncWebServer
}

func NewApiTester() *apiTester {
	s := NewFileSyncWebServer("config.yaml", false)
	s.bindApiHandlers(&s.engine.RouterGroup)
	at := new(apiTester)
	at.server = s
	at.ts = httptest.NewTLSServer(at.server.engine)
	at.responseRecord = httptest.NewRecorder()
	return at
}
func (tester *apiTester) Do(method, path string, setRequest func(*http.Request)) *http.Response {
	req := httptest.NewRequest(method, tester.ts.URL+path, nil)
	w := httptest.NewRecorder()
	tester.server.engine.ServeHTTP(w, req)
	return w.Result()
}
func init() {
	os.Chdir("../")
}
