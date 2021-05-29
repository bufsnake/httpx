package screenshot

type screenshot interface {
	Run() (string, error)
}

func NewScreenShot(url string, timeout int, path string) screenshot {
	return &chrome{url: url, timeout: timeout * 6, path: path}
}
