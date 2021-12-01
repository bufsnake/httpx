package config

import (
	"fmt"
	"github.com/chromedp/cdproto/fetch"
	"sync"
)

// terminal options
type Terminal struct {
	Probes            map[string]map[string]bool // all probe data
	API               string                     // http server
	Target            string                     // single target
	Targets           string                     // multiple targets
	Threads           int                        // scan threads
	Proxy             string                     // proxy
	HeadlessProxy     string                     // headless proxy
	Timeout           int                        // http request timeout
	ChromePath        string                     // screenshot chrome path
	Output            string                     // output file，default .html
	Path              string                     // URL Path
	DisableScreenshot bool                       // disable screenshot
	DisableHeadless   bool                       // disable headless
	DisplayError      bool                       // Show error
	AllowJump         bool                       // allow jump
	Silent            bool                       // silent model
	Server            bool                       // server model
	CIDR              string                     // CIDR file
	Stop              *bool
	ProbesL           sync.Mutex
	GetPath           bool // 获取请求的二级、三级、目录
	GetUrl            bool // 获取请求URL，包括参数
	Port              Ports
	Header            Header
	Headers           []*fetch.HeaderEntry
	Method            string // 请求方式
	Data              string // 请求体
	Rebuild           bool   // 重新生成datas表
	OutOfRange        map[string]bool

	// 保存资产清单
	AssetsL sync.Mutex
	Assets  map[string]bool
}

func (c *Terminal) AddAssets(url string) {
	c.AssetsL.Lock()
	defer c.AssetsL.Unlock()
	c.Assets[url] = true
}

func (c *Terminal) IsExist(url string) bool {
	c.AssetsL.Lock()
	defer c.AssetsL.Unlock()
	if _, ok := c.Assets[url]; !ok {
		return false
	}
	return true
}

type Ports []string

// Value ...
func (i *Ports) String() string {
	return fmt.Sprint(*i)
}

// Set 方法是flag.Value接口, 设置flag Value的方法.
// 通过多个flag指定的值， 所以我们追加到最终的数组上.
func (i *Ports) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Header []string

func (i *Header) String() string {
	return fmt.Sprint(*i)
}

func (i *Header) Set(value string) error {
	*i = append(*i, value)
	return nil
}
