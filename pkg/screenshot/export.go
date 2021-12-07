package screenshot

import (
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/wappalyzer"
)

type Screenshot interface {
	Run(url string) (string, string, string, map[string]bool, map[string]wappalyzer.Technologie, map[string]bool, error)
	InitEnv() error
	Cancel()
	SwitchTab()
}

func NewScreenShot(conf *config.Terminal, l *log.Log) Screenshot {
	return &chrome{timeout: conf.Timeout * 6, conf_: conf, l: l}
}
