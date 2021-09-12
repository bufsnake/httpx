package requests

import (
	"fmt"
	"github.com/bufsnake/httpx/pkg/log"
	"sync"
)

type httpx struct {
	url       string
	proxy     string
	timeout   int
	URLS      []*request
	lock      sync.Mutex
	log       *log.Log
	logerror  bool
	allowjump bool
}

func (h *httpx) Run() error {
	httpv := "http://" + h.url
	httpvs := "https://" + h.url
	http := NewRequest(httpv, h.proxy, h.timeout, h.allowjump)
	https := NewRequest(httpvs, h.proxy, h.timeout, h.allowjump)
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
		if err != nil && h.logerror {
			fmt.Println("\r", err)
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
		if err != nil && h.logerror {
			fmt.Println("\r", err)
		}
	}()
	wghttpx.Wait()
	return nil
}
