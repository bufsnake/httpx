package requests

import (
	"bytes"
	"crypto/tls"
	"github.com/bufsnake/httpx/pkg/useragent"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"html"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type request struct {
	url         string
	status_code int
	title       string
	length      int
	timeout     int
	http_dump   string
	proxy       string
	allowjump   bool
}

func (r *request) Run() error {
	cli := http.Client{
		Timeout: time.Duration(r.timeout) * time.Second,
	}
	if !r.allowjump {
		cli.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	transport := http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}
	if r.proxy != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(r.proxy)
		}
		transport.Proxy = proxy
	}
	cli.Transport = &transport
	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", useragent.RandomUserAgent())
	req.Header.Set("Connection", "close")
	do, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer do.Body.Close()
	resp, err := httputil.DumpResponse(do, true)
	if err != nil {
		return err
	}
	r.http_dump = string(resp)
	body, err := ioutil.ReadAll(do.Body)
	if err != nil {
		return err
	}
	r.length = len(body)
	r.title, err = extracttitle(string(body))
	if err != nil {
		r.title = ""
	}
	for i := 0; i < 3; i++ {
		r.title = strings.TrimLeft(r.title, " ")
		r.title = strings.TrimLeft(r.title, "\t")
		r.title = strings.TrimRight(r.title, " ")
		r.title = strings.TrimRight(r.title, "\t")
	}
	r.status_code = do.StatusCode
	return nil
}

func (r *request) GetUrl() string {
	return r.url
}

func (r *request) GetStatusCode() int {
	return r.status_code
}

func (r *request) GetTitle() string {
	return r.title
}

func (r *request) GetLength() int {
	return r.length
}

func (r *request) GetHTTPDump() string {
	return r.http_dump
}

// 获取网站标题
func extracttitle(body string) (string, error) {
	title := ""
	var re = regexp.MustCompile(`(?im)<\s*title.*>(.*?)<\s*/\s*title>`)
	for _, match := range re.FindAllString(body, -1) {
		title = html.UnescapeString(trimTitleTags(match))
		break
	}
	if !utf8.Valid([]byte(title)) {
		reader := transform.NewReader(bytes.NewReader([]byte(title)), simplifiedchinese.GBK.NewDecoder())
		d, err := ioutil.ReadAll(reader)
		if err != nil {
			return title, err
		}
		return string(d), nil
	}
	return title, nil
}

func trimTitleTags(title string) string {
	titleBegin := strings.Index(title, ">")
	titleEnd := strings.Index(title, "</")
	return title[titleBegin+1 : titleEnd]
}
