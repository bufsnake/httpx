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
	"github.com/bufsnake/parseip"
	"github.com/gin-gonic/gin"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
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

func main() {
	conf := config.Terminal{}
	flag.StringVar(&conf.Target, "target", "", "single target, example:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.StringVar(&conf.Targets, "targets", "", "multiple goals, examlpe:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.IntVar(&conf.Threads, "thread", 10, "config probe thread")
	flag.StringVar(&conf.Proxy, "proxy", "", "config probe proxy, example: http://127.0.0.1:8080")
	flag.IntVar(&conf.Timeout, "timeout", 10, "config probe http request timeout")
	flag.StringVar(&conf.Output, "output", time.Now().Format("200601021504")+".db", "output database file name")
	flag.StringVar(&conf.Path, "path", "", "specify request path for probe or screenshot")
	flag.StringVar(&conf.ChromePath, "chrome-path", "", "chrome browser path")
	flag.StringVar(&conf.HeadlessProxy, "headless-proxy", "", "chrome browser proxy")
	flag.StringVar(&conf.CIDR, "cidr", "", "cidr file, example:\n127.0.0.1\n127.0.0.5-20\n127.0.0.2-127.0.0.20\n127.0.0.1/18")
	flag.BoolVar(&conf.DisableScreenshot, "disable-screenshot", false, "disable screenshot")
	flag.BoolVar(&conf.DisplayError, "display-error", false, "display error")
	flag.BoolVar(&conf.AllowJump, "allow-jump", false, "allow jump")
	flag.BoolVar(&conf.Silent, "silent", false, "silent output")
	flag.BoolVar(&conf.Server, "server", false, "read the database by starting the web service")
	flag.Parse()

	database, err := modelsImpl.NewDatabase(conf.Output)
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

	probes := make(map[string]bool)
	file := make([]byte, 0)
	if conf.Target != "" {
		probes[conf.Target] = true
	} else if conf.Targets != "" {
		file, err = os.ReadFile(conf.Targets)
		if err != nil {
			fmt.Println(err)
			return
		}
		urls := strings.Split(string(file), "\n")
		for i := 0; i < len(urls); i++ {
			urls[i] = strings.Trim(urls[i], "\r")
			if urls[i] == "" {
				continue
			}
			probes[urls[i]] = true
		}
	} else if conf.CIDR != "" {
		file, err = os.ReadFile(conf.CIDR)
		if err != nil {
			fmt.Println(err)
			return
		}
		urls := strings.Split(string(file), "\n")
		for i := 0; i < len(urls); i++ {
			urls[i] = strings.Trim(urls[i], "\r")
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
				probes[parseip.UInt32ToIP(ip)] = true
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
		group.GET("/search", newAPI.Search)
		group.GET("/copy", newAPI.Copy)
		err = engine.Run(":9100")
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			probes[sc.Text()] = true
		}
		if err = sc.Err(); err != nil {
			fmt.Println("failed to read input:", err)
			return
		}
	}

	if conf.CIDR == "" {
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
			probes[temp.Host] = true
		}
	}

	if len(probes) == 0 {
		flag.Usage()
		return
	}

	var p float64 = 0
	l := log.Log{
		Percentage:   &p,
		AllHTTP:      float64(len(probes) * 2),
		Conf:         conf,
		StartTime:    time.Now(),
		Silent:       conf.Silent,
		DB:           &database,
		DisplayError: conf.DisplayError,
	}
	go l.Bar()

	conf.Probes = probes

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	stop := false
	conf.Stop = &stop

	// error: concurrent map iteration and map write
	// Stop Flag
	go func() {
		select {
		case _ = <-c:
			*conf.Stop = true
			conf.ProbesL.Lock()
			defer conf.ProbesL.Unlock()
			output := ""
			for un, _ := range conf.Probes {
				output += un + "\n"
			}
			_ = os.WriteFile("unprobe_assets_"+time.Now().Format("200601021504"), []byte(strings.Trim(output, "\n")), 0777)
			os.Exit(1)
		}
	}()

	newCore := core.NewCore(&l, conf)
	err = newCore.Probe()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !l.Silent {
		fmt.Println()
	}
}
