package requests

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"sync"
)

type httpx struct {
	conf *config.Terminal
	url  string
	URLS []*request
	lock sync.Mutex
	l    *log.Log
}

func (h *httpx) Run() error {
	httpv := "http://" + h.url
	httpvs := "https://" + h.url
	http := NewRequest(httpv, h.conf, h.l)
	https := NewRequest(httpvs, h.conf, h.l)
	wghttpx := sync.WaitGroup{}
	wghttpx.Add(2)
	go func() {
		defer wghttpx.Done()
		err := http.Run()
		if err != nil {
			h.l.Error(err)
			return
		}
		h.lock.Lock()
		h.URLS = append(h.URLS, http)
		h.lock.Unlock()
	}()
	go func() {
		defer wghttpx.Done()
		err := https.Run()
		if err != nil {
			h.l.Error(err)
			return
		}
		h.lock.Lock()
		h.URLS = append(h.URLS, https)
		h.lock.Unlock()
	}()
	wghttpx.Wait()
	return nil
}
