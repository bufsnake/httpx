package screenshot

import "github.com/bufsnake/httpx/config"

type Screenshot interface {
	Run(url string) (string, error)
	InitEnv() error
	Cancel()
	SwitchTab()
}

func NewScreenShot(timeout int, path string, conf config.Terminal) Screenshot {
	return &chrome{timeout: timeout * 6, path: path, conf_: conf}
}
