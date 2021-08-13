package requests

import "github.com/bufsnake/httpx/pkg/log"

func NewHttpx(url, proxy string, timeout int, l log.Log, logerror, allowjump bool) *httpx {
	return &httpx{url: url, proxy: proxy, timeout: timeout, log: l, logerror: logerror, allowjump: allowjump}
}

func NewRequest(url, proxy string, timeout int, allowjump bool) *request {
	return &request{url: url, proxy: proxy, timeout: timeout, allowjump: allowjump}
}
