package core

import (
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/requests"
	"github.com/bufsnake/httpx/pkg/screenshot"
	. "github.com/logrusorgru/aurora"
	"strconv"
	"strings"
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
	ss := screenshot.NewScreenShot(c.conf)
	err := ss.InitEnv()
	if err != nil {
		return err
	}
	defer ss.Cancel()
	go ss.SwitchTab()
	for i := 0; i < c.conf.Threads; i++ {
		urlwait.Add(1)
		go func(w *sync.WaitGroup, u chan string, l log.Log, c config.Terminal, screen_shot screenshot.Screenshot) {
			defer w.Done()
			for t := range u {
				httpx := requests.NewHttpx(strings.Trim(t, "/")+"/"+strings.TrimLeft(c.URI, "/"), c.Proxy, c.Timeout, l, c.DisplayError, c.AllowJump)
				err = httpx.Run()
				if err != nil {
					l.Println(err)
					continue
				}
				if c.DisableScreenshot {
					for j := 0; j < len(httpx.URLS); j++ {
						if c.Search != "" {
							if strings.Contains(httpx.URLS[j].GetHTTPDump(), c.Search) {
								l.Println("["+BrightGreen(strconv.Itoa(httpx.URLS[j].GetStatusCode())).String()+"]", "["+BrightWhite(httpx.URLS[j].GetUrl()).String()+"]", "["+BrightRed(strconv.Itoa(httpx.URLS[j].GetLength())).String()+"]", "["+BrightCyan(httpx.URLS[j].GetTitle()).String()+"]", "["+BrightBlue(time.Now().Format("2006-01-02 15:04:05")).String()+"]")
							}
						} else {
							l.Println("["+BrightGreen(strconv.Itoa(httpx.URLS[j].GetStatusCode())).String()+"]", "["+BrightWhite(httpx.URLS[j].GetUrl()).String()+"]", "["+BrightRed(strconv.Itoa(httpx.URLS[j].GetLength())).String()+"]", "["+BrightCyan(httpx.URLS[j].GetTitle()).String()+"]", "["+BrightBlue(time.Now().Format("2006-01-02 15:04:05")).String()+"]")
						}
						l.PercentageAdd()
					}
				} else {
					for j := 0; j < len(httpx.URLS); j++ {
						output := false
						if c.Search != "" {
							if strings.Contains(httpx.URLS[j].GetHTTPDump(), c.Search) {
								output = true
							}
						} else {
							output = true
						}
						if output {
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
							run, err := screen_shot.Run(strings.Trim(httpx.URLS[j].GetUrl(), "/") + "/" + strings.TrimLeft(c.URI, "/"))
							if err == nil {
								data.Image = run
							} else if err != nil && c.DisplayError {
								fmt.Println("\r", err)
							}
							l.Println("["+BrightGreen(data.StatusCode).String()+"]", "["+BrightWhite(data.URL).String()+"]", "["+BrightRed(data.BodyLength).String()+"]", "["+BrightCyan(data.Title).String()+"]", "["+BrightBlue(data.CreateTime).String()+"]")
							l.OutputHTML(data)
						}
						l.PercentageAdd()
					}
				}
				for j := 0; j < 2-len(httpx.URLS); j++ {
					l.PercentageAdd()
				}
			}
		}(&urlwait, urlchan, c.log, c.conf, ss)
	}
	for url, _ := range c.conf.Probes {
		urlchan <- url
	}
	close(urlchan)
	urlwait.Wait()
	return nil
}
