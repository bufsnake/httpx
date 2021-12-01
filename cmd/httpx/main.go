package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/internal/api"
	"github.com/bufsnake/httpx/internal/core"
	"github.com/bufsnake/httpx/internal/modelsImpl"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/utils"
	"github.com/bufsnake/httpx/pkg/wappalyzer"
	"github.com/bufsnake/parseip"
	"github.com/chromedp/cdproto/fetch"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

func init() {
	// 开启多核模式
	runtime.GOMAXPROCS(runtime.NumCPU() * 3 / 4)
	// 关闭 GIN Debug模式
	// 设置工具可打开的文件描述符
	var rLimit syscall.Rlimit
	rLimit.Max = 999999
	rLimit.Cur = 999999
	if runtime.GOOS == "darwin" {
		rLimit.Cur = 10240
	}
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_ = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}

//go:embed website
var website embed.FS

//go:embed wappalyzer
var wappalyzer_fs embed.FS

// 列wappalyzer_fs目录，没找到_.json
//go:embed wappalyzer/src/technologies/_.json
var file_ string

func main() {
	err := wappalyzer.InitWappalyzerDB(wappalyzer_fs, file_)
	if err != nil {
		fmt.Println(err)
		return
	}
	wappalyzer.SetReadICONURL("/v1/geticon?icon=")
	//newWappalyzer := wappalyzer.NewWappalyzer()
	//newWappalyzer.Headers(map[string]string{
	//	"Server": "Darwin",
	//})
	//newWappalyzer.DNS(os.Args[1])
	//os.Exit(1)
	conf := config.Terminal{}
	conf.Assets = make(map[string]bool)
	// default content-type
	err = conf.Header.Set("Content-Type: application/x-www-form-urlencoded")
	if err != nil {
		fmt.Println(err)
		return
	}
	flag.StringVar(&conf.Target, "target", "", "single target, example:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.StringVar(&conf.Targets, "targets", "", "multiple goals, examlpe:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.IntVar(&conf.Threads, "thread", 10, "config probe thread")
	flag.StringVar(&conf.Proxy, "proxy", "", "config probe proxy, example: http://127.0.0.1:8080")
	flag.StringVar(&conf.API, "api", "127.0.0.1:9100", "http server listen address")
	flag.IntVar(&conf.Timeout, "timeout", 10, "config probe http request timeout")
	flag.StringVar(&conf.Output, "output", time.Now().Format("200601021504"), "output database file name")
	flag.StringVar(&conf.Path, "path", "", "specify request path for probe or screenshot")
	flag.StringVar(&conf.ChromePath, "chrome-path", "", "chrome browser path")
	flag.StringVar(&conf.HeadlessProxy, "headless-proxy", "", "chrome browser proxy")
	flag.StringVar(&conf.CIDR, "cidr", "", "cidr file, example:\n127.0.0.1\n127.0.0.5-20\n127.0.0.2-127.0.0.20\n127.0.0.1/18")
	flag.Var(&conf.Port, "port", "specify port, example:\n-port 80 -port 8080")
	flag.BoolVar(&conf.DisableScreenshot, "disable-screenshot", false, "disable screenshot")
	flag.BoolVar(&conf.GetPath, "get-path", false, "get all request path")
	flag.BoolVar(&conf.DisableHeadless, "disable-headless", false, "disable chrome headless")
	flag.BoolVar(&conf.GetUrl, "get-url", false, "get all request url")
	flag.BoolVar(&conf.DisplayError, "display-error", false, "display error")
	flag.BoolVar(&conf.AllowJump, "allow-jump", false, "allow jump")
	flag.BoolVar(&conf.Silent, "silent", false, "silent output")
	flag.BoolVar(&conf.Rebuild, "rebuild", false, "rebuild data table")
	flag.BoolVar(&conf.Server, "server", false, "read the database by starting the web service")

	flag.Var(&conf.Header, "header", "specify request header, example:\n-header 'Content-Type: application/json' -header 'Bypass: 127.0.0.1'")
	flag.StringVar(&conf.Method, "method", "GET", "request method, example:\n-method GET")
	flag.StringVar(&conf.Data, "data", "", "request body data, example:\n-data 'test=test'")
	flag.Parse()

	if strings.HasSuffix(conf.Output, ".db") {
		conf.Output = strings.ReplaceAll(conf.Output, ".db", "")
	}

	database, err := modelsImpl.NewDatabase(conf.Output+".db", conf.Server && conf.Rebuild)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = database.InitDatabase()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if conf.Proxy != "" {
		_, err = url.Parse(conf.Proxy)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	probes := make(map[string]map[string]bool)
	file := make([]byte, 0)
	if conf.Target != "" {
		probes[conf.Target] = make(map[string]bool)
	} else if conf.Targets != "" {
		file, err = os.ReadFile(conf.Targets)
		if err != nil {
			fmt.Println(err)
			return
		}
		urls := strings.Split(string(file), "\n")
		for i := 0; i < len(urls); i++ {
			urls[i] = strings.Trim(urls[i], " \t\r")
			if urls[i] == "" {
				continue
			}
			probes[urls[i]] = make(map[string]bool)
		}
	} else if conf.CIDR != "" {
		if len(conf.Port) == 0 {
			_ = conf.Port.Set("80")
		}
		file, err = os.ReadFile(conf.CIDR)
		if err != nil {
			fmt.Println(err)
			return
		}
		urls := strings.Split(string(file), "\n")
		for i := 0; i < len(urls); i++ {
			urls[i] = strings.Trim(urls[i], " \t\r")
			if urls[i] == "" {
				continue
			}
			var (
				start uint32
				end   uint32
			)
			start, end, err = parseip.ParseIP(urls[i])
			if err != nil {
				fmt.Println(urls[i], err)
				continue
			}
			for ip := start; ip <= end; ip++ {
				if _, ok := probes[parseip.UInt32ToIP(ip)]; !ok {
					probes[parseip.UInt32ToIP(ip)] = make(map[string]bool)
				}
			}
		}
	} else if conf.Server {
		var website_t fs.FS
		website_t, err = fs.Sub(website, "website")
		if err != nil {
			fmt.Println(err)
			return
		}
		newAPI := api.NewAPI(&database)
		engine := gin.Default()
		engine.StaticFS("/ui", http.FS(website_t))
		engine.NoRoute(func(c *gin.Context) {
			c.Redirect(301, "/ui")
		})
		group := engine.Group("/v1")
		group.GET("/getdatas", newAPI.GetData)
		group.GET("/imageload", newAPI.ImageLoad)
		group.POST("/search", newAPI.Search)
		group.POST("/copy", newAPI.Copy)
		group.GET("/fingers", newAPI.FingerLoad)
		group.GET("/geticon", newAPI.GetICON)
		err = engine.Run(conf.API)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			probes[sc.Text()] = nil
		}
		if err = sc.Err(); err != nil {
			fmt.Println("failed to read input:", err)
			return
		}
	}

	if conf.CIDR == "" && len(conf.Port) != 0 {
		fmt.Println("can not specify port, only CIDR work")
		return
	}

	if (conf.GetPath || conf.GetUrl) && conf.DisableScreenshot {
		fmt.Println("get path/get url must enable screenshot")
		return
	}

	if conf.CIDR == "" {
		temp_probes := make(map[string]map[string]bool)
		for probe, _ := range probes {
			temp := &url.URL{}
			if strings.HasPrefix(probe, "http://") || strings.HasPrefix(probe, "https://") {
				temp, err = url.Parse(probe)
			} else {
				temp, err = url.Parse("http://" + probe)
			}
			delete(probes, probe)
			if err != nil {
				fmt.Println(probe, err)
				continue
			}
			if strings.Contains(temp.Host, ":") {
				port := strings.Split(temp.Host, ":")[1]
				temp.Host = strings.Split(temp.Host, ":")[0]
				if port != "443" && port != "80" {
					temp.Host = temp.Host + ":" + port
				}
			}
			if _, ok := temp_probes[temp.Host]; !ok {
				temp_probes[temp.Host] = make(map[string]bool)
			}
			if temp.Path == "" {
				temp.Path = "/"
			}
			if temp.RawQuery != "" {
				temp.Path = temp.Path + "?" + temp.RawQuery
			}
			temp_probes[temp.Host][temp.Path] = true
		}
		probes = temp_probes
	}

	if len(probes) == 0 {
		flag.Usage()
		return
	}

	var total_probe = 0
	for _, paths := range probes {
		if len(paths) == 0 {
			total_probe += 1
		}
		total_probe += len(paths)
	}
	if len(conf.Port) != 0 {
		total_probe *= len(conf.Port)
	}

	var p float64 = 0

	conf.Probes = probes
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	conf.OutOfRange = make(map[string]bool)

	for i := 0; i < len(config.OutOfRange); i++ {
		conf.OutOfRange[config.OutOfRange[i]] = true
	}

	for asset, _ := range conf.Probes {
		if !utils.IsDomain(asset) {
			continue
		}
		if strings.Contains(asset, ":") {
			asset = strings.Split(asset, ":")[0]
		}
		if _, ok := conf.OutOfRange[asset]; ok {
			delete(conf.OutOfRange, asset)
		}
	}

	stop := false
	conf.Stop = &stop

	wait := sync.WaitGroup{}
	// error: concurrent map iteration and map write
	// Stop Flag
	go func() {
		select {
		case _ = <-c:
			wait.Add(1)
			defer wait.Done()
			*conf.Stop = true
			conf.ProbesL.Lock()
			defer conf.ProbesL.Unlock()
			content := ""
			for unprobe, _ := range conf.Probes {
				content += unprobe + "\n"
			}
			content = strings.Trim(content, "\n")
			if content == "" {
				return
			}
			err = os.WriteFile(conf.Output+"_unprobe", []byte(strings.Trim(content, "\n")), 0777)
			if err != nil {
				fmt.Println("\n" + err.Error())
			}
		}
	}()

	l := log.Log{
		Percentage:     &p,
		NumberOfAssets: len(probes),
		AllHTTP:        float64(total_probe * 2),
		Conf:           &conf,
		StartTime:      time.Now(),
		Silent:         conf.Silent,
		DB:             &database,
		DisplayError:   conf.DisplayError,
	}
	go l.Bar()

	for i := 0; i < len(conf.Header); i++ {
		key_val := strings.SplitN(conf.Header[i], ":", 2)
		if len(key_val) != 2 {
			fmt.Println("warning: header error", conf.Header[i])
			continue
		}
		if strings.ToUpper(strings.Trim(key_val[0], " ")) == "BYPASS" {
			bypass := []string{"Forwarded", "Forwarded-For", "Forwarded-For-Ip", "X-Client-IP", "X-Custom-IP-Authorization", "X-Forward", "X-Forwarded", "X-Forwarded-By", "X-Forwarded-For", "X-Forwarded-For-Original", "X-Forwarded-Server", "X-Forwared-Host", "X-HTTP-Host-Override", "X-Host", "X-Originating-IP", "X-Remote-Addr", "X-Remote-IP"}
			for b := 0; b < len(bypass); b++ {
				conf.Headers = append(conf.Headers, &fetch.HeaderEntry{bypass[b], strings.Trim(key_val[1], " ")})
			}
			continue
		}
		conf.Headers = append(conf.Headers, &fetch.HeaderEntry{strings.Trim(key_val[0], " "), strings.Trim(key_val[1], " ")})
	}

	newCore := core.NewCore(&l, &conf)
	err = newCore.Probe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !l.Silent {
		fmt.Println()
	}
	wait.Wait()
}
