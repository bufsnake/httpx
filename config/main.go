package config

import "sync"

// terminal options
type Terminal struct {
	Probes            map[string]bool // all probe data
	Target            string          // single target
	Targets           string          // multiple targets
	Threads           int             // scan threads
	Proxy             string          // proxy
	HeadlessProxy     string          // headless proxy
	Timeout           int             // http request timeout
	ChromePath        string          // screenshot chrome path
	Output            string          // output file，default .html
	Path              string          // URL Path
	DisableScreenshot bool            // disable screenshot
	DisplayError      bool            // Show error
	AllowJump         bool            // allow jump
	Silent            bool            // silent model
	Server            bool            // server model
	CIDR              string          // CIDR file
	Stop              *bool
	ProbesL           sync.Mutex
}
