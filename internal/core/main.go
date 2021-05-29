package core

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/requests"
	"github.com/bufsnake/httpx/pkg/screenshot"
	. "github.com/logrusorgru/aurora"
	"strconv"
	"sync"
	"time"
)

type Core struct {
	log  log.Log
	conf config.Terminal
}

func NewCore(l log.Log, c config.Terminal) Core {
	return Core{log: l, conf: c}
}

func (c *Core) Probe() error {
	urlchan := make(chan string, c.conf.Threads)
	urlwait := sync.WaitGroup{}
	for i := 0; i < c.conf.Threads; i++ {
		urlwait.Add(1)
		go func(w *sync.WaitGroup, u chan string, l log.Log, c config.Terminal) {
			defer w.Done()
			for t := range u {
				httpx := requests.NewHttpx(t, c.Proxy, c.Timeout, l)
				err := httpx.Run()
				if err != nil {
					l.Println(err)
					continue
				}
				for j := 0; j < len(httpx.URLS); j++ {
					data := config.OutputData{
						ID:         1,
						URL:        httpx.URLS[j].GetUrl(),
						Title:      httpx.URLS[j].GetTitle(),
						StatusCode: strconv.Itoa(httpx.URLS[j].GetStatusCode()),
						BodyLength: strconv.Itoa(httpx.URLS[j].GetLength()),
						CreateTime: time.Now().Format("2006-01-02 15:04:05"),
						Image:      "",
						HTTPDump:   httpx.URLS[j].GetHTTPDump(),
					}
					screen_shot := screenshot.NewScreenShot(httpx.URLS[j].GetUrl(), c.Timeout, c.ChromePath)
					run, err := screen_shot.Run()
					if err != nil {
						l.Println(err)
					} else {
						data.Image = run
					}
					l.Println("["+BrightGreen(data.StatusCode).String()+"]", "["+BrightWhite(data.URL).String()+"]", "["+BrightRed(data.BodyLength).String()+"]", "["+BrightCyan(data.Title).String()+"]", "["+BrightBlue(data.CreateTime).String()+"]")
					l.OutputHTML(data)
					l.PercentageAdd()
				}
				for j := 0; j < 2-len(httpx.URLS); j++ {
					l.PercentageAdd()
				}
			}
		}(&urlwait, urlchan, c.log, c.conf)
	}
	for url, _ := range c.conf.Probes {
		urlchan <- url
	}
	close(urlchan)
	urlwait.Wait()
	return nil
}
