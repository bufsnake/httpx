package screenshot

type Screenshot interface {
	Run(url string) (string, error)
	InitEnv() error
	Cancel()
	SwitchTab()
}

func NewScreenShot(timeout int, path string) Screenshot {
	return &chrome{timeout: timeout * 6, path: path}
}
