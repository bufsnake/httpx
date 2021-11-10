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

func main() {
	conf := config.Terminal{}
	flag.StringVar(&conf.Target, "target", "", "single target, example:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.StringVar(&conf.Targets, "targets", "", "multiple goals, examlpe:\n127.0.0.1\n127.0.0.1:8080\nhttp://127.0.0.1")
	flag.IntVar(&conf.Threads, "thread", 10, "config probe thread")
	flag.StringVar(&conf.Proxy, "proxy", "", "config probe proxy, example: http://127.0.0.1:8080")
	flag.IntVar(&conf.Timeout, "timeout", 10, "config probe http request timeout")
	flag.StringVar(&conf.Output, "output", time.Now().Format("200601021504"), "output database file name")
	flag.StringVar(&conf.Path, "path", "", "specify request path for probe or screenshot")
	flag.StringVar(&conf.ChromePath, "chrome-path", "", "chrome browser path")
	flag.StringVar(&conf.HeadlessProxy, "headless-proxy", "", "chrome browser proxy")
	flag.StringVar(&conf.CIDR, "cidr", "", "cidr file, example:\n127.0.0.1\n127.0.0.5-20\n127.0.0.2-127.0.0.20\n127.0.0.1/18")
	flag.Var(&conf.Port, "port", "specify port, example:\n-port 80 -port 8080")
	flag.Var(&conf.Header, "H", "specify request header, example:\n-H 'Content-Type: application/json' -H 'Bypass: 127.0.0.1'")
	flag.StringVar(&conf.Method, "X", "GET", "request method")
	flag.StringVar(&conf.Data, "D", "", "request body data")
	flag.BoolVar(&conf.DisableScreenshot, "disable-screenshot", false, "disable screenshot")
	flag.BoolVar(&conf.GetPath, "get-path", false, "get all request path")
	flag.BoolVar(&conf.GetUrl, "get-url", false, "get all request url")
	flag.BoolVar(&conf.DisplayError, "display-error", false, "display error")
	flag.BoolVar(&conf.AllowJump, "allow-jump", false, "allow jump")
	flag.BoolVar(&conf.Silent, "silent", false, "silent output")
	flag.BoolVar(&conf.Rebuild, "rebuild", false, "rebuild data table")
	flag.BoolVar(&conf.Server, "server", false, "read the database by starting the web service")
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
		err = engine.Run(":9100")
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
	outofrange := []string{
		"12333.gov.cn",
		"12348.gov.cn",
		"12388.gov.cn",
		"131002.net",
		"1688.com",
		"1daas.com",
		"360.cn",
		"360safe.com",
		"5-studio.com",
		"51eds.com",
		"591adb.com",
		"66163.com",
		"720yuntu.com",
		"8686c.com",
		"adobe.com",
		"ag-grid.com",
		"agrtc.cn",
		"aishu.cn",
		"ajax.googleapis.com",
		"aliapp.org",
		"alibaba.com",
		"alibabagroup.com",
		"alicdn.com",
		"alipay.com",
		"alipayobjects.com",
		"aliyun.com",
		"aliyuncs.com",
		"amap.com",
		"ant.design",
		"anyrtc.io",
		"apache.org",
		"apachecn.org",
		"apachefriends.org",
		"apachehaus.com",
		"apih5.com",
		"apple.com",
		"arcgisonline.com",
		"atlassian.com",
		"autonavi.com",
		"avatars.githubusercontent.com",
		"avuejs.com",
		"axure.com",
		"azm.workers.dev",
		"bankcomm.com",
		"bcebos.com",
		"bdimg.com",
		"bdstatic.com",
		"beian.gov.cn",
		"beijing.gov.cn",
		"beisen.com",
		"bing.com",
		"bit.ly",
		"bizcharts.net",
		"bjbxzjxh.org.cn",
		"bjd.com.cn",
		"bocionline.com",
		"bohemiancoding.com",
		"bookan.com.cn",
		"bookstack.cn",
		"bootcdn.net",
		"bootcss.com",
		"bootstrapcdn.com",
		"bt.cn",
		"c114.com.cn",
		"caac.gov.cn",
		"cankaoxiaoxi.com",
		"cbirc.gov.cn",
		"ccgrp.com.cn",
		"ccps.gov.cn",
		"cct.google",
		"ccxi.com.cn",
		"cdn-go.cn",
		"centos.org",
		"cesium.com",
		"chceb.com",
		"chinabidding.com.cn",
		"chinacloudapi.cn",
		"chinatax.gov.cn",
		"cib.com.cn",
		"cipa.jp",
		"citrix.com",
		"clickhouse.yandex",
		"cloudbility.com",
		"cloudera.com",
		"cloudflare.com",
		"cma.gov.cn",
		"cnipa.gov.cn",
		"cnjiwang.com",
		"cnki.net",
		"cnsphoto.com",
		"commonmark.org",
		"conac.cn",
		"corporateshowcase.com",
		"court.gov.cn",
		"cq.gov.cn",
		"crbimds.cn",
		"crccib.com",
		"creativecommons.org",
		"crecgec.com",
		"crgtt.com",
		"crmg-ec.com.cn",
		"crphdm.com",
		"csrc.gov.cn",
		"customs.gov.cn",
		"cyol.com",
		"czbank.com",
		"daiwei.org",
		"datagrand.com",
		"datatables.net",
		"dcloud.net.cn",
		"debian.org",
		"deloitte.com",
		"devexpress.com",
		"dingtalk.com",
		"dji.com",
		"do1.com.cn",
		"douban.com",
		"dousha8ao.com",
		"dowsure.com",
		"e-cology.com.cn",
		"e-shrailway.com",
		"earthsdk.com",
		"eastmoney.com",
		"easy-mock.com",
		"ebuypress.com",
		"edgly.net",
		"eisoo.com",
		"elemecdn.com",
		"emqx.cn",
		"emqx.io",
		"esgcc.com.cn",
		"eso.org",
		"esri.com",
		"etouch.cn",
		"euroland.com",
		"eurolandir.com",
		"example.com",
		"excalidraw.com",
		"facebook.com",
		"facebook.net",
		"fangchan.com",
		"fanruan.com",
		"faqrobot.com",
		"faqrobot.org",
		"fastfish.com",
		"fb.me",
		"financialnews.com.cn",
		"finereport.com",
		"fmprc.gov.cn",
		"fonts.googleapis.com",
		"forestry.gov.cn",
		"fwxgx.com",
		"gansudaily.com.cn",
		"gdtv.cn",
		"gethue.com",
		"getpocket.com",
		"gettyimages.com",
		"ggj.gov.cn",
		"git.io",
		"gitee.com",
		"gitee.io",
		"github.com",
		"github.io",
		"githubassets.com",
		"gitlab.com",
		"gitter.im",
		"gjj12329.cn",
		"gjzwfw.gov.cn",
		"glodon.com",
		"glodonedu.com",
		"gmdaily.cn",
		"gmpg.org",
		"gmw.cn",
		"gnu.org",
		"gogs.io",
		"goharbor.io",
		"google.cn",
		"google.com",
		"google.com.hk",
		"googleapis.com",
		"googletagmanager.com",
		"grafana.com",
		"grafana.org",
		"graphite.readthedocs.io",
		"gravatar.com",
		"gstatic.com",
		"gtimg.cn",
		"gtimg.com",
		"gtja.com",
		"gzdaily.cn",
		"h3yun.com",
		"hainan.gov.cn",
		"handsontable.com",
		"hanweb.com",
		"hatena.ne.jp",
		"hbzwfw.gov.cn",
		"hcharts.cn",
		"hebnews.cn",
		"helloweba.net",
		"helm.sh",
		"hhhtmetro.com",
		"highcharts.com",
		"highcharts.com.cn",
		"hikvision.com",
		"hitcounter.pythonanywhere.com",
		"hizom.cn",
		"hlj.gov.cn",
		"hotjar.com",
		"htsec.com",
		"huawei.com",
		"hubeidaily.net",
		"hust.edu.cn",
		"ibm.com",
		"idqqimg.com",
		"ietf.org",
		"inkscape.org",
		"instagram.com",
		"instapaper.com",
		"iptc.org",
		"irasia.com",
		"java.com",
		"jcy.gov.cn",
		"jiandaoyun.com",
		"jiaohuo.net",
		"jiathis.com",
		"jinshixun.com",
		"jinshuju.com",
		"jinshuju.net",
		"jinshujuapp.com",
		"jinshujufiles.com",
		"jl.gov.cn",
		"jlntv.cn",
		"jnwtv.com",
		"jsdelivr.net",
		"json-schema.org",
		"justep.com",
		"jxzwfww.gov.cn",
		"kaipuyun.cn",
		"kjur.github.io",
		"kubernetes.io",
		"kubesphere.io",
		"kuboard.cn",
		"laravel.com",
		"launchpad.net",
		"layui.com",
		"lenovo.com",
		"line.me",
		"linkedin.com",
		"linktbm.com",
		"list-manage.com",
		"live.com",
		"localking.com.tw",
		"logrocket.com",
		"logrocket.io",
		"loli.net",
		"longtailvideo.com",
		"macromedia.com",
		"magi.com",
		"mapbox.cn",
		"mapbox.com",
		"mapinfo.com",
		"mapking.com",
		"maps.googleapis.com",
		"maxcdn.com",
		"mca.gov.cn",
		"mct.gov.cn",
		"mee.gov.cn",
		"metabase.com",
		"microsoft.com",
		"miit.gov.cn",
		"min.io",
		"mixcloud.com",
		"mnr.gov.cn",
		"moa.gov.cn",
		"mocky.io",
		"moe.gov.cn",
		"mof.gov.cn",
		"mofcom.gov.cn",
		"mohrss.gov.cn",
		"mohurd.gov.cn",
		"moj.gov.cn",
		"momentjs.com",
		"most.gov.cn",
		"mot.gov.cn",
		"mozilla.com",
		"mozilla.github.io",
		"mozilla.org",
		"mps.gov.cn",
		"mva.gov.cn",
		"mwr.gov.cn",
		"mxhichina.com",
		"mybank.cn",
		"myqcloud.com",
		"mysite.com",
		"ncexc.com",
		"ncha.gov.cn",
		"ndrc.gov.cn",
		"nea.gov.cn",
		"news.cn",
		"nhc.gov.cn",
		"nhsa.gov.cn",
		"nia.gov.cn",
		"nist.gov",
		"njtu.edu.cn",
		"nlark.com",
		"nmgggfw.cn",
		"nmpa.gov.cn",
		"noembed.com",
		"npmjs.org",
		"nr-data.net",
		"nra.gov.cn",
		"nrta.gov.cn",
		"nuget.org",
		"number-7.cn",
		"oasis-open.org",
		"ocks.org",
		"oclc.org",
		"ogp.me",
		"opengis.net",
		"openid.net",
		"openoffice.org",
		"openstreetmap.org",
		"operamasks.org",
		"oracle.com",
		"oscca.gov.cn",
		"oschina.net",
		"panjiachen.github.io",
		"paytm.in",
		"pbc.gov.cn",
		"phoenixtv.com",
		"picsum.photos",
		"pinterest.com",
		"placekitten.com",
		"plot.ly",
		"polyfill.io",
		"polyv.net",
		"prismic.io",
		"prometheus.io",
		"purl.org",
		"pytorch.org",
		"qcloud.com",
		"qnssl.com",
		"qq.com",
		"qqmail.com",
		"quanshi.com",
		"raintank.io",
		"raw.githubusercontent.com",
		"reachstar.com",
		"reactjs.org",
		"readthedocs.org",
		"redhat.com",
		"renren.com",
		"ruoyi.vip",
		"s3.amazonaws.com",
		"safe.gov.cn",
		"samr.gov.cn",
		"sangfor.com.cn",
		"sangfor.net",
		"sasac.gov.cn",
		"sastind.gov.cn",
		"sctv.com",
		"seajs.org",
		"sentry.io",
		"sh.gov.cn",
		"shaanxi.gov.cn",
		"shiseido.co.jp",
		"showdoc.cc",
		"sina.cn",
		"sina.com.cn",
		"sinaimg.cn",
		"sinajs.cn",
		"siques.cn",
		"slack.com",
		"smartclient.com",
		"sobot.com",
		"sonarsource.com",
		"sonatype.com",
		"soperson.com",
		"soundcloud.com",
		"sourceforge.net",
		"southcn.com",
		"spb.gov.cn",
		"sport.gov.cn",
		"stackoverflow.com",
		"stamen-tiles.a.ssl.fastly.net",
		"staoedu.com",
		"statecharts.io",
		"stats.gov.cn",
		"stdaily.com",
		"streamable.com",
		"stumbleupon.com",
		"sun.com",
		"suo.im",
		"supermap.com.cn",
		"supermapol.com",
		"swagger.io",
		"swfobject.googlecode.com",
		"sxrb.com",
		"sxzwfw.gov.cn",
		"sysdsoft.cn",
		"t.me",
		"tabix.io",
		"talk99.cn",
		"taobao.com",
		"taobao.org",
		"taobaocdn.com",
		"tfhub.dev",
		"thymeleaf.org",
		"tianditu.com",
		"tianditu.gov.cn",
		"tianqi.com",
		"tianqistatic.com",
		"tielu.cn",
		"tiny.cloud",
		"tjyun.com",
		"tobacco.gov.cn",
		"toutiao.com",
		"transloadit.com",
		"trs.cn",
		"trustutn.org",
		"tumblr.com",
		"twitch.tv",
		"twitter.com",
		"twxrrd.com",
		"typicode.com",
		"udesk.cn",
		"umijs.org",
		"unpkg.com",
		"url.cn",
		"verdaccio.org",
		"vicp.cc",
		"videodelivery.net",
		"vidyard.com",
		"vimeo.com",
		"virtualearth.net",
		"vueadmin.cn",
		"vuejs-templates.github.io",
		"vuejs.org",
		"w3.org",
		"w3c.org",
		"weather.com.cn",
		"weaver.com.cn",
		"webgl.org",
		"webrtc.org",
		"webtrn.cn",
		"weibo.cn",
		"weibo.com",
		"wikipedia.org",
		"wistia.com",
		"www.gov.cn",
		"xa.gov.cn",
		"xiaoe-tech.com",
		"xinhuanet.com",
		"xinjiang.gov.cn",
		"xinnet.com",
		"xjbt.gov.cn",
		"xmlsoap.org",
		"xmpp.org",
		"xunlei.com",
		"yahooapis.com",
		"ycwb.com",
		"yn.gov.cn",
		"youku.com",
		"youtube.com",
		"youtucc.com",
		"ys7.com",
		"ytimg.com",
		"yuansci.cn",
		"yuanxinghy.com",
		"yunaq.com",
		"yunmd.net",
		"yunnan.cn",
		"yunshipei.com",
		"yunteams.cn",
		"yzcdn.cn",
		"zabbix.com",
		"zencdn.net",
		"zentao.net",
		"zhaopin.com",
		"zhiye.com",
		"zhongguowangshi.com",
		"zpert.com",
		"zplan.cc",
		"zqrb.cn",
		"zt178.cn",
	}
	for i := 0; i < len(outofrange); i++ {
		conf.OutOfRange[outofrange[i]] = true
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
