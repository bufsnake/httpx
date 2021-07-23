package requests

import (
	"github.com/bufsnake/httpx/pkg/log"
	"strings"
	"sync"
)

type httpx struct {
	url     string
	proxy   string
	timeout int
	URLS    []*request
	lock    sync.Mutex
	log     log.Log
}

func (h *httpx) Run() error {
	httpv := "http://" + h.url
	httpvs := "https://" + h.url
	urlsp := strings.Split(h.url, ":")
	if len(urlsp) == 2 {
		if urlsp[1] == "80" {
			httpv = "http://" + urlsp[0]
		} else if urlsp[1] == "443" {
			httpvs = "https://" + urlsp[0]
		}
	}
	http := NewRequest(httpv, h.proxy, h.timeout)
	https := NewRequest(httpvs, h.proxy, h.timeout)
	wghttpx := sync.WaitGroup{}
	wghttpx.Add(2)
	go func() {
		defer wghttpx.Done()
		err := http.Run()
		if err == nil {
			h.lock.Lock()
			h.URLS = append(h.URLS, http)
			h.lock.Unlock()
		}
	}()
	go func() {
		defer wghttpx.Done()
		err := https.Run()
		if err == nil {
			h.lock.Lock()
			h.URLS = append(h.URLS, https)
			h.lock.Unlock()
		}
	}()
	wghttpx.Wait()
	return nil
}
