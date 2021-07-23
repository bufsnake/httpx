package screenshot

import "github.com/bufsnake/httpx/config"

type Screenshot interface {
	Run(url string) (string, error)
	InitEnv() error
	Cancel()
	SwitchTab()
}

func NewScreenShot(conf config.Terminal) Screenshot {
	return &chrome{timeout: conf.Timeout * 6, conf_: conf}
}
