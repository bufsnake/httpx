package core

import (
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
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
	log  *log.Log
	conf config.Terminal
}

func NewCore(l *log.Log, c config.Terminal) Core {
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
		go c.routine(&urlwait, urlchan, ss)
	}
	for url, _ := range c.conf.Probes {
		urlchan <- url
	}
	close(urlchan)
	urlwait.Wait()
	return nil
}

func (c *Core) routine(w *sync.WaitGroup, u chan string, screen_shot screenshot.Screenshot) {
	defer w.Done()
	for t := range u {
		httpx := requests.NewHttpx(strings.Trim(t, "/")+"/"+strings.TrimLeft(c.conf.URI, "/"), c.conf.Proxy, c.conf.Timeout, c.log, c.conf.DisplayError, c.conf.AllowJump)
		err := httpx.Run()
		if err != nil {
			c.log.Println(err)
			continue
		}
		if c.conf.DisableScreenshot {
			for j := 0; j < len(httpx.URLS); j++ {
				if c.conf.Search != "" {
					if strings.Contains(httpx.URLS[j].GetHTTPDump(), c.conf.Search) {
						c.log.Println("["+BrightGreen(strconv.Itoa(httpx.URLS[j].GetStatusCode())).String()+"]", "["+BrightWhite(httpx.URLS[j].GetUrl()).String()+"]", "["+BrightRed(strconv.Itoa(httpx.URLS[j].GetLength())).String()+"]", "["+BrightCyan(httpx.URLS[j].GetTitle()).String()+"]", "["+BrightBlue(time.Now().Format("2006-01-02 15:04:05")).String()+"]")
					}
				} else {
					c.log.Println("["+BrightGreen(strconv.Itoa(httpx.URLS[j].GetStatusCode())).String()+"]", "["+BrightWhite(httpx.URLS[j].GetUrl()).String()+"]", "["+BrightRed(strconv.Itoa(httpx.URLS[j].GetLength())).String()+"]", "["+BrightCyan(httpx.URLS[j].GetTitle()).String()+"]", "["+BrightBlue(time.Now().Format("2006-01-02 15:04:05")).String()+"]")
				}
				c.log.PercentageAdd()
			}
		} else {
			for j := 0; j < len(httpx.URLS); j++ {
				output := false
				if c.conf.Search != "" {
					if strings.Contains(httpx.URLS[j].GetHTTPDump(), c.conf.Search) {
						output = true
					}
				} else {
					output = true
				}
				if output {
					data := models.Datas{
						URL:        httpx.URLS[j].GetUrl(),
						Title:      httpx.URLS[j].GetTitle(),
						StatusCode: strconv.Itoa(httpx.URLS[j].GetStatusCode()),
						BodyLength: strconv.Itoa(httpx.URLS[j].GetLength()),
						CreateTime: time.Now().Format("2006-01-02 15:04:05"),
						Image:      "",
						HTTPDump:   httpx.URLS[j].GetHTTPDump(),
						TLS:        httpx.URLS[j].GetTLS(),
						ICP:        httpx.URLS[j].GetICP(),
					}
					run, icp, err := screen_shot.Run(strings.Trim(httpx.URLS[j].GetUrl(), "/") + "/" + strings.TrimLeft(c.conf.URI, "/"))
					if err == nil {
						data.Image = run
						if icp != "" && icp != data.ICP {
							data.ICP += "|" + icp
						}
					} else if err != nil && c.conf.DisplayError {
						fmt.Println("\r", err)
					}
					c.log.Println("["+BrightGreen(data.StatusCode).String()+"]", "["+BrightWhite(data.URL).String()+"]", "["+BrightRed(data.BodyLength).String()+"]", "["+BrightCyan(data.Title).String()+"]", "["+BrightBlue(data.CreateTime).String()+"]")
					c.log.OutputHTML(data)
				}
				c.log.PercentageAdd()
			}
		}
		for j := 0; j < 2-len(httpx.URLS); j++ {
			c.log.PercentageAdd()
		}
	}
}
