package screenshot

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/wappalyzer"
)

type Screenshot interface {
	Run(url string, XFrameOptions string) (string, string, string, map[string]wappalyzer.Technologie, error)
	InitEnv() error
	Cancel()
	SwitchTab()
}

func NewScreenShot(conf *config.Terminal, l *log.Log) Screenshot {
	return &chrome{timeout: conf.Timeout * 6, conf_: conf, l: l}
}
