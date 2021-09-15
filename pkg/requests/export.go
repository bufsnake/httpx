package requests

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
)

func NewHttpx(url string, conf *config.Terminal, l *log.Log) *httpx {
	return &httpx{url: url, conf: conf, l: l}
}

func NewRequest(url string, conf *config.Terminal, l *log.Log) *request {
	return &request{url: url, conf: conf, l: l}
}
