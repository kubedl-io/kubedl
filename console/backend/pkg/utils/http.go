package utils

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// getClient is get a default httpClient
func getClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		DisableCompression: true,
		DisableKeepAlives:  true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	return client
}

// RequestWithPost is a http client with url, header and params
func RequestWithPost(reqUrl string, header map[string]string, params map[string]string) (status int, body string, err error) {
	return RequestWithHeader(http.MethodPost, reqUrl, header, params)
}

// RequestWithHeader is a http client with header, url, and params
func RequestWithHeader(method string, reqUrl string, header map[string]string, params map[string]string) (status int, body string, err error) {

	client := getClient()
	paramsA := ""
	if method == http.MethodPost {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		paramsA = values.Encode()
	}

	req, _ := http.NewRequest(method, reqUrl, strings.NewReader(paramsA))

	for k, v := range header {
		req.Header.Set(k, v)
	}

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if method == http.MethodGet {
		values := req.URL.Query()
		for k, v := range params {
			values.Add(k, v)
		}
		req.URL.RawQuery = values.Encode()
	}

	getResp, err := client.Do(req)
	if err != nil {
		return status, body, err
	}
	defer getResp.Body.Close()
	status = getResp.StatusCode
	getBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return status, body, err
	}

	decodeBytes, err := url.QueryUnescape(string(getBody))
	if err != nil {
		return status, body, err
	}
	return status, string(decodeBytes), nil
}
