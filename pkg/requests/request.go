package requests

import (
	"bytes"
	"crypto/tls"
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
	timeout     int
	http_dump   string
	proxy       string
	allowjump   bool
	tls         string
	icp         string
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
	r.title = strings.Trim(r.title, " \t\r\n")
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
