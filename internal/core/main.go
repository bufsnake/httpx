package core

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/models"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/requests"
	"github.com/bufsnake/httpx/pkg/screenshot"
	"github.com/bufsnake/wappalyzer"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Core struct {
	log  *log.Log
	conf *config.Terminal
	spl  sync.Mutex
}

func NewCore(l *log.Log, c *config.Terminal) Core {
	return Core{log: l, conf: c}
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
			for path := range paths {
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
	return nil
}

func parsePath(host, path string) map[string]bool {
	reqs := make(map[string]bool)
	subpaths := strings.Split(path, "/")
	paths_ := make([]string, 0)
	for i := 0; i < len(subpaths); i++ {
		if len(strings.Trim(subpaths[i], " \\")) == 0 {
			continue
		}
		paths_ = append(paths_, strings.Trim(subpaths[i], " \\"))
	}
	subpaths = paths_
	reqs[host+"/"] = true
	pathsFunc(host, subpaths, 0, &reqs)
	return reqs
}

func pathsFunc(host string, path []string, index int, reqs *map[string]bool) {
	if index == len(path) {
		return
	}
	if strings.Contains(path[index], ".") && index == len(path)-1 {
		return
	}
	(*reqs)[host+"/"+strings.Join(path[:index+1], "/")+"/"] = true
	index++
	pathsFunc(host, path, index, reqs)
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
				URL:           strings.Trim(httpx.URLS[j].GetUrl(), "/ "),
				Title:         httpx.URLS[j].GetTitle(),
				StatusCode:    strconv.Itoa(httpx.URLS[j].GetStatusCode()),
				BodyLength:    strconv.Itoa(httpx.URLS[j].GetLength()),
				CreateTime:    time.Now().Format("2006-01-02 15:04:05"),
				Image:         "",
				HTTPDump:      httpx.URLS[j].GetHTTPDump(),
				TLS:           httpx.URLS[j].GetTLS(),
				ICP:           httpx.URLS[j].GetICP(),
				XFrameOptions: httpx.URLS[j].GetXFrameOptions(),
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
		qr := make(map[string]bool)
		if !c.conf.DisableScreenshot {
			var (
				image string
				icp   string
				title string
				err   error
			)
			fingers := make(map[string]wappalyzer.Technologie)
			image, title, fingers, err = screen_shot.Run(data.URL)
			if err != nil {
				c.log.Error(err)
			} else {
				data.Image = image
				if icp != "" && icp != data.ICP {
					data.ICP += "|" + icp
				}
			}
			if title != "" {
				data.Title = title
			}
			if len(fingers) != 0 {
				fingers_ := make([]models.Finger, 0)
				for name, val := range fingers {
					Categories := ""
					for i := 0; i < len(val.Categories); i++ {
						Categories += val.Categories[i].Name + "\n"
					}
					fingers_ = append(fingers_, models.Finger{
						URL:        data.URL,
						Name:       name,
						Confidence: val.Confidence,
						Version:    val.Version,
						ICON:       val.Icon,
						WebSite:    val.Website,
						CPE:        val.Cpe,
						Categories: strings.Trim(Categories, "\n"),
					})
				}
				c.log.SaveFinger(fingers_)
			}
		}
		c.log.Println(data.StatusCode, data.URL, data.BodyLength, data.Title, data.CreateTime, qr)
		c.log.SaveData(data)
		c.log.PercentageAdd()
	}
}
