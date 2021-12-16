package requests

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/useragent"
	"github.com/bufsnake/httpx/pkg/utils"
	"github.com/grantae/certinfo"
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
	http_dump   string
	tls         string
	icp         string
	conf        *config.Terminal
	l           *log.Log
}

func (r *request) Run() error {
	cli := http.Client{
		Timeout: time.Duration(r.conf.Timeout) * time.Second,
	}
	if !r.conf.AllowJump {
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
	if r.conf.Proxy != "" {
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(r.conf.Proxy)
		}
		transport.Proxy = proxy
	}
	cli.Transport = &transport
	methods := []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace}

	r.conf.Method = strings.ToUpper(r.conf.Method)
	if !strings.Contains(strings.Join(methods, " "), r.conf.Method) {
		return errors.New("unsupport method " + r.conf.Method)
	}

	body_data := strings.NewReader("")
	if r.conf.Data != "" {
		body_data = strings.NewReader(r.conf.Data)
	}

	r.conf.AddAssets(r.url)

	req, err := http.NewRequest(r.conf.Method, r.url, body_data)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", useragent.RandomUserAgent())
	req.Header.Set("Cookie", "rememberMe=Lisan")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "close")
	for i := 0; i < len(r.conf.Headers); i++ {
		r.conf.Headers[i].Value = strings.ReplaceAll(r.conf.Headers[i].Value, "{{RAND}}", utils.RandString(10))
		if strings.ToUpper(r.conf.Headers[i].Name) == "HOST" {
			req.Host = r.conf.Headers[i].Value
			continue
		}
		req.Header.Set(r.conf.Headers[i].Name, r.conf.Headers[i].Value)
	}
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
	var convert bool
	r.title, convert, err = extracttitle(string(body))
	if err != nil {
		r.title = ""
	}
	r.title = strings.Trim(r.title, " \t\r\n")
	if convert {
		reader := transform.NewReader(bytes.NewReader(resp), simplifiedchinese.GBK.NewDecoder())
		resp, err = ioutil.ReadAll(reader)
		if err == nil {
			r.http_dump = string(resp)
		}
	}

	if r.title == "400 The plain HTTP request was sent to HTTPS port" {
		return errors.New("response title is '400 The plain HTTP request was sent to HTTPS port'")
	}
	if r.title == "400 No required SSL certificate was sent" {
		return errors.New("response title is '400 The plain HTTP request was sent to HTTPS port'")
	}
	if strings.Contains(r.http_dump, "This combination of host and port requires TLS.") {
		return errors.New("response title is '400 The plain HTTP request was sent to HTTPS port'")
	}

	r.status_code = do.StatusCode
	if do.TLS != nil {
		certChain := do.TLS.PeerCertificates
		tls_ := "================================================================== "
		for j := 0; j < len(certChain); j++ {
			cert := certChain[j]
			result, err := certinfo.CertificateText(cert)
			if err != nil {
				continue
			}
			tls_ += result
			tls_ += "================================================================== "
		}
		tls_ += "End"
		r.tls = tls_
	}
	r.icp = utils.ICPInfo(r.http_dump)
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

func (r *request) GetTLS() string {
	return r.tls
}

func (r *request) GetHTTPDump() string {
	return r.http_dump
}

func (r *request) GetICP() string {
	return r.icp
}

// 获取网站标题 - ret title,是否转换,error
func extracttitle(body string) (string, bool, error) {
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
			return title, false, err
		}
		return string(d), true, nil
	}
	return title, false, nil
}

func trimTitleTags(title string) string {
	titleBegin := strings.Index(title, ">")
	titleEnd := strings.Index(title, "</")
	return title[titleBegin+1 : titleEnd]
}
