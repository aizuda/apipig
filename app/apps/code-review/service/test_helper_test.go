package service

import (
	"net/http"
	"strings"
)

func newFiberRequest(path, body string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "http://localhost"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
