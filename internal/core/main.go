package core

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/requests"
	"github.com/bufsnake/httpx/pkg/screenshot"
	"github.com/bufsnake/httpx/pkg/utils"
	"github.com/weppos/publicsuffix-go/publicsuffix"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Core struct {
	log    *log.Log
	conf   *config.Terminal
	reqs   []map[string]bool
	reqs_l sync.Mutex
}

func NewCore(l *log.Log, c *config.Terminal) Core {
	reqs := make([]map[string]bool, 0)
	return Core{log: l, conf: c, reqs: reqs}
}

func (c *Core) Probe() error {
	urlchan := make(chan [2]string, 200)
	urlwait := sync.WaitGroup{}
	var ss screenshot.Screenshot
	if !c.conf.DisableScreenshot {
		ss = screenshot.NewScreenShot(c.conf, c.log)
		err := ss.InitEnv()
		if err != nil {
			return err
		}
		defer ss.Cancel()
		go ss.SwitchTab()
	}
	datas := make(chan models.Datas, 100)
	for i := 0; i < 2*c.conf.Threads; i++ {
		urlwait.Add(1)
		go c.routine(&urlwait, urlchan, datas)
	}
	screenshot_wait := sync.WaitGroup{}
	for i := 0; i < c.conf.Threads; i++ {
		screenshot_wait.Add(1)
		go c.screenshot(&screenshot_wait, datas, ss)
	}

	c.conf.ProbesL.Lock()
	defer c.conf.ProbesL.Unlock()
	for host, paths := range c.conf.Probes {
		if *c.conf.Stop {
			break
		}
		if c.conf.CIDR == "" {
			for path, _ := range paths {
				urlchan <- [2]string{host, path}
			}
			delete(c.conf.Probes, host)
			continue
		}
		flag := true
		for i := 0; i < len(c.conf.Port); i++ {
			if c.conf.Port[i] == "80" || c.conf.Port[i] == "443" {
				if flag {
					flag = false
					urlchan <- [2]string{host, "/"}
				}
			} else {
				urlchan <- [2]string{host + ":" + c.conf.Port[i], "/"}
			}
		}
		delete(c.conf.Probes, host)
	}
	close(urlchan)

	urlwait.Wait()
	close(datas)
	screenshot_wait.Wait()
	if c.conf.GetPath {
		getpath := ""
		geturl := ""
		for _, req := range c.reqs {
			for urlstr, _ := range req {
				// 判断是否在黑名单
				parse, err := url.Parse(urlstr)
				if err != nil {
					c.log.Error(urlstr, err)
					continue
				}
				if strings.Contains(parse.Host, ":") {
					parse.Host = strings.Split(parse.Host, ":")[0]
				}
				if utils.IsDomain(parse.Host) {
					domain, err := publicsuffix.Domain(parse.Host)
					if err == nil {
						if _, ok := c.conf.OutOfRange[domain]; ok {
							continue
						}
					}
				}
				if c.conf.GetUrl {
					geturl += urlstr + "\n"
				}
				subpaths := parsePath(parse.Scheme+"://"+parse.Host, parse.Path)
				for subpath := range subpaths {
					getpath += subpath + "\n"
				}
			}
		}

		getpath = strings.Trim(getpath, "\n")
		if getpath != "" {
			_ = os.WriteFile(c.conf.Output+"_path", []byte(strings.Trim(getpath, "\n")), 0777)
		}
		geturl = strings.Trim(geturl, "\n")
		if geturl != "" {
			_ = os.WriteFile(c.conf.Output+"_url", []byte(strings.Trim(geturl, "\n")), 0777)
		}
	}
	return nil
}

func parsePath(host, path string) map[string]bool {
	reqs := make(map[string]bool)
	subpaths := strings.Split(path, "/")
	for i := 0; i < len(subpaths); i++ {
		if subpaths[i] == "" || strings.Contains(subpaths[i], ".") {
			continue
		}
		subpath := "/"
		for j := 0; j <= i; j++ {
			if subpaths[j] == "" || strings.Contains(subpaths[j], ".") {
				continue
			}
			subpath += subpaths[j] + "/"
		}
		reqs[host+subpath] = true
	}
	return reqs
}

func (c *Core) routine(w *sync.WaitGroup, u chan [2]string, datas chan models.Datas) {
	defer w.Done()
	for t := range u {
		path := ""
		if strings.Trim(t[1], "/ \t") != "" {
			path = "/" + strings.Trim(t[1], "/")
		}
		if strings.Trim(c.conf.Path, "/ \t") != "" {
			path += "/" + strings.TrimLeft(c.conf.Path, "/")
		}
		httpx := requests.NewHttpx(strings.Trim(t[0], "/")+path, c.conf, c.log)
		err := httpx.Run()
		if err != nil {
			c.log.Error(err)
			c.log.PercentageAdd()
			c.log.PercentageAdd()
			continue
		}
		for j := 0; j < len(httpx.URLS); j++ {
			data := models.Datas{
				URL:        strings.Trim(httpx.URLS[j].GetUrl(), "/ "),
				Title:      httpx.URLS[j].GetTitle(),
				StatusCode: strconv.Itoa(httpx.URLS[j].GetStatusCode()),
				BodyLength: strconv.Itoa(httpx.URLS[j].GetLength()),
				CreateTime: time.Now().Format("2006-01-02 15:04:05"),
				Image:      "",
				HTTPDump:   httpx.URLS[j].GetHTTPDump(),
				TLS:        httpx.URLS[j].GetTLS(),
				ICP:        httpx.URLS[j].GetICP(),
			}
			datas <- data
		}
		for j := 0; j < 2-len(httpx.URLS); j++ {
			c.log.PercentageAdd()
		}
	}
}

func (c *Core) screenshot(w *sync.WaitGroup, datas chan models.Datas, screen_shot screenshot.Screenshot) {
	defer w.Done()
	for data := range datas {
		if !c.conf.DisableScreenshot {
			var (
				run   string
				icp   string
				title string
				err   error
			)
			reqs := make(map[string]bool)
			// Get Path from JS Files
			run, icp, title, reqs, err = screen_shot.Run(data.URL)
			if err != nil {
				c.log.Error(err)
			} else {
				data.Image = run
				if icp != "" && icp != data.ICP {
					data.ICP += "|" + icp
				}
			}
			if title != "" {
				data.Title = title
			}
			c.reqs_l.Lock()
			c.reqs = append(c.reqs, reqs)
			c.reqs_l.Unlock()
		}
		c.log.Println(data.StatusCode, data.URL, data.BodyLength, data.Title, data.CreateTime)
		c.log.SaveData(data)
		c.log.PercentageAdd()
	}
}
