package internal

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
)

func SendRestApiRequest(method string, access_token string, urlPath string, body []byte, skip_tls_verify bool) []byte {
	headers := map[string][]string{
		"Content-Type":  []string{"application/x-www-form-urlencoded"},
		"Accept":        []string{"application/json"},
		"Authorization": []string{"Bearer " + access_token},
	}
	req, err := http.NewRequest(method, urlPath, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.Header = headers

	tlsConfig := tls.Config{}
	tlsConfig.InsecureSkipVerify = skip_tls_verify
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return b
}
