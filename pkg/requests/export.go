package requests

import "github.com/bufsnake/httpx/pkg/log"

func NewHttpx(url, proxy string, timeout int, l log.Log, logerror bool) *httpx {
	return &httpx{url: url, proxy: proxy, timeout: timeout, log: l, logerror: logerror}
}

func NewRequest(url, proxy string, timeout int) *request {
	return &request{url: url, proxy: proxy, timeout: timeout}
}
