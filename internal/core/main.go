package core

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/requests"
	"github.com/bufsnake/httpx/pkg/screenshot"
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
		delete(c.conf.Probes, url)
	}
	close(urlchan)
	urlwait.Wait()
	return nil
}

func (c *Core) routine(w *sync.WaitGroup, u chan string, screen_shot screenshot.Screenshot) {
	defer w.Done()
	for t := range u {
		httpx := requests.NewHttpx(strings.Trim(t, "/")+"/"+strings.TrimLeft(c.conf.Path, "/"), c.conf.Proxy, c.conf.Timeout, c.log, c.conf.DisplayError, c.conf.AllowJump)
		err := httpx.Run()
		if err != nil {
			c.log.Error(err)
			c.log.PercentageAdd()
			c.log.PercentageAdd()
			continue
		}
		for j := 0; j < len(httpx.URLS); j++ {
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
			if !c.conf.DisableScreenshot {
				var (
					run string
					icp string
				)
				run, icp, err = screen_shot.Run(strings.Trim(httpx.URLS[j].GetUrl(), "/") + "/" + strings.TrimLeft(c.conf.Path, "/"))
				if err != nil {
					c.log.Error(err)
				} else {
					data.Image = run
					if icp != "" && icp != data.ICP {
						data.ICP += "|" + icp
					}
				}
			}
			c.log.Println(data.StatusCode, data.URL, data.BodyLength, data.Title, data.CreateTime)
			c.log.SaveData(data)
			c.log.PercentageAdd()
		}
		for j := 0; j < 2-len(httpx.URLS); j++ {
			c.log.PercentageAdd()
		}
	}
}
