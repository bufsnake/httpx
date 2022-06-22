package screenshot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/log"
	"github.com/bufsnake/httpx/pkg/useragent"
	"github.com/bufsnake/httpx/pkg/utils"
	"github.com/bufsnake/wappalyzer"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type chrome struct {
	timeout int
	ctx     context.Context
	cancel  context.CancelFunc
	conf_   *config.Terminal
	l       *log.Log
}

var bypass_headless_detect = `(function(w, n, wn) {
  // Pass the Webdriver Test.
  Object.defineProperty(n, 'webdriver', {
    get: () => false,
  });

  // Pass the Plugins Length Test.
  // Overwrite the plugins property to use a custom getter.
  Object.defineProperty(n, 'plugins', {
    // This just needs to have length > 0 for the current test,
    // but we could mock the plugins too if necessary.
    get: () => [1, 2, 3, 4, 5],
  });

  // Pass the Languages Test.
  // Overwrite the plugins property to use a custom getter.
  Object.defineProperty(n, 'languages', {
    get: () => ['en-US', 'en'],
  });

  // Pass the Chrome Test.
  // We can mock this in as much depth as we need for the test.
  w.chrome = {
    runtime: {},
  };

  // Pass the Permissions Test.
  const originalQuery = wn.permissions.query;
  return wn.permissions.query = (parameters) => (
    parameters.name === 'notifications' ?
      Promise.resolve({ state: Notification.permission }) :
      originalQuery(parameters)
  );
})(window, navigator, window.navigator);`

// image, title, body, fingers, err
func (c *chrome) Run(url string, XFrameOptions string) (string, string, string, map[string]wappalyzer.Technologie, error) {
	buf, title, body, fingers, err := c.run_chromedp(url, XFrameOptions)
	if err != nil {
		return "", title, body, nil, err
	}
	return base64.StdEncoding.EncodeToString(buf), title, body, fingers, nil
}

// Init Start CTX
func (c *chrome) InitEnv() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !c.conf_.DisableHeadless),
		chromedp.Flag("incognito", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.UserAgent(useragent.RandomUserAgent()),
		chromedp.WindowSize(1920, 1080),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.NoDefaultBrowserCheck,
		chromedp.NoFirstRun,
		chromedp.Flag("proxy-bypass-list", "<-loopback>"),
	)
	if c.conf_.HeadlessProxy != "" {
		fmt.Println(c.conf_.HeadlessProxy)
		opts = append(opts, chromedp.ProxyServer(c.conf_.HeadlessProxy))
	}
	if c.conf_.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(c.conf_.ChromePath))
	}
	c.ctx, c.cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	c.ctx, c.cancel = chromedp.NewContext(c.ctx)
	err := chromedp.Run(c.ctx, page.Close())
	if err != nil {
		return errors.New("Init Start Chrome Error: " + err.Error())
	}
	return nil
}

// switch tabs, auto close tab
func (c *chrome) SwitchTab() {
	targets_ := make(map[target.ID]time.Time)
	for {
		targets, err := chromedp.Targets(c.ctx)
		if err != nil {
			c.l.Error(err)
			continue
		}
		c.l.SetNumberOfTabs(len(targets))
		for i := 0; i < len(targets); i++ {
			if _, ok := targets_[targets[i].TargetID]; !ok {
				targets_[targets[i].TargetID] = time.Now()
			}
			err = target.ActivateTarget(targets[i].TargetID).Do(cdp.WithExecutor(c.ctx, chromedp.FromContext(c.ctx).Browser))
			if err != nil {
				c.l.Error(err)
				continue
			}
			if time.Now().Sub(targets_[targets[i].TargetID]).Seconds() >= float64(30*time.Second) {
				err = target.CloseTarget(targets[i].TargetID).Do(cdp.WithExecutor(c.ctx, chromedp.FromContext(c.ctx).Browser))
				if err != nil {
					c.l.Error(err)
				}
			}
		}
		time.Sleep(1 * time.Second / 5)
	}
}

// End Start CTX
func (c *chrome) Cancel() {
	defer c.cancel()
}

// 监听
func (c *chrome) listen(ctx context.Context) func(ev interface{}) {
	compile := regexp.MustCompile("(setTimeout\\(|setInterval\\(|Function\\(|alert\\(|eval\\(|\\.write\\(|\\.writeln\\(|\\.createComment\\(|\\.createTextNode\\(|\\.createElement\\(|\\.innerHTML|\\.className|\\.innerText|\\.textContent|\\.title|\\.href)")
	return func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			// Add Headers
			// Can Not Set Host Header
			if len(c.conf_.Headers) == 0 || !c.conf_.IsExist(strings.Trim(e.Request.URL, "/")) {
				go func() {
					err := fetch.ContinueRequest(e.RequestID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Target))
					if err != nil {
						c.l.Error(err)
					}
				}()
			} else {
				go func() {
					headers := make([]*fetch.HeaderEntry, 0)
					for i := 0; i < len(c.conf_.Headers); i++ {
						if strings.ToUpper(c.conf_.Headers[i].Name) == "HOST" {
							continue
						}
						c.conf_.Headers[i].Value = strings.ReplaceAll(c.conf_.Headers[i].Value, "{{RAND}}", utils.RandString(10))
						headers = append(headers, c.conf_.Headers[i])
					}
					err := fetch.ContinueRequest(e.RequestID).
						WithURL(e.Request.URL).
						WithMethod(c.conf_.Method).
						WithPostData(base64.StdEncoding.EncodeToString([]byte(c.conf_.Data))). // If set, overrides the post data in the request. (Encoded as a base64 string when passed over JSON)
						WithHeaders(headers).
						Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Target))
					if err != nil {
						c.l.Error(err)
					}
				}()
			}
		case *page.EventJavascriptDialogOpening:
			// Disable JavaScriptDialog
			// chrome IWA 不影响截图
			t := page.HandleJavaScriptDialog(false)
			go func() {
				if err := chromedp.Run(ctx, t); err != nil {
					c.l.Error(errors.New("\nrunChromedp error: " + err.Error()))
				}
			}()
		case *runtime.EventConsoleAPICalled:
			for i := 0; i < len(e.Args); i++ {
				var val string
				err := json.Unmarshal(e.Args[i].Value, &val)
				if err != nil {
					continue
				}
				if !strings.HasPrefix(val, "bufsnake ") {
					continue
				}
				if !compile.MatchString(val) {
					continue
				}
				val = strings.ReplaceAll(val, "bufsnake ", "==== ")
				err = utils.AppendFile(c.conf_.Output+"_postMessage.txt", val+"\n")
				if err != nil {
					c.l.Error(err)
				}
			}
		}
	}
}

func (c *chrome) run_chromedp(urlstr string, XFrameOptions string) ([]byte, string, string, map[string]wappalyzer.Technologie, error) {
	var (
		buf   []byte
		title string
		body  string
	)
	newContext, cancelFunc := chromedp.NewContext(c.ctx)
	defer cancelFunc()
	newContext, cancelFunc = context.WithTimeout(newContext, 30*time.Second)
	defer cancelFunc()
	newWappalyzer := wappalyzer.NewWappalyzer(c.l.DisplayError)
	parse, err := url.Parse(urlstr)
	if err != nil {
		return nil, "", "", nil, err
	}
	if strings.Contains(parse.Host, ":") {
		parse.Host = strings.Split(parse.Host, ":")[0]
	}
	//newWappalyzer.DetectDNS(parse.Host)
	//newWappalyzer.DetectRobots(urlstr)
	chromedp.ListenTarget(newContext, c.listen(newContext))
	chromedp.ListenTarget(newContext, newWappalyzer.DetectListen(newContext))
	if err = chromedp.Run(newContext, c.screenshot(urlstr, &buf, &title, &body, newWappalyzer.DetectActions(), XFrameOptions)); err != nil {
		return []byte{}, "", "", nil, errors.New("chromedp.run error: " + err.Error())
	}
	return buf, title, body, newWappalyzer.GetFingers(), nil
}

func (c *chrome) screenshot(urlstr string, res *[]byte, title, body *string, fingeractions chromedp.Action, XFrameOptions string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(c.conf_.Headers) > 0 {
				err := fetch.Enable().Do(ctx)
				if err != nil {
					return err
				}
			}
			_, err := page.AddScriptToEvaluateOnNewDocument(bypass_headless_detect).Do(ctx)
			if err != nil {
				return errors.New(fmt.Sprintf("AddScriptToEvaluateOnNewDocument %s", err))
			}
			err = chromedp.Navigate(urlstr).Do(ctx)
			if err != nil {
				// 401 验证问题
				if !strings.Contains(err.Error(), "page load error net::ERR_INVALID_AUTH_CREDENTIALS") {
					errs := target.CloseTarget(chromedp.FromContext(ctx).Target.TargetID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
					if errs != nil {
						return errors.New(fmt.Sprintf("Navigate CloseTarget %s", err))
					}
					return errors.New(fmt.Sprintf("Navigate Target Error %s", err))
				}
			}
			return nil
		}),
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, err := page.CaptureScreenshot().
				WithQuality(80).
				WithFormat("png").
				WithFromSurface(true).
				WithCaptureBeyondViewport(true).
				WithClip(&page.Viewport{
					X:      0,
					Y:      0,
					Width:  1920,
					Height: 1000,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return errors.New("captureScreenshot error: " + err.Error())
			}
			*res = buf
			return nil
		}),
		// 执行获取postMessage监听函数
		chromedp.ActionFunc(func(ctx context.Context) error {
			XFrameOptions = strings.ToUpper(XFrameOptions)
			if XFrameOptions == "DENY" || XFrameOptions == "SAMEORIGIN" {
				return nil
			}
			var r = &runtime.RemoteObject{}
			err := chromedp.EvaluateAsDevTools(fmt.Sprintf(`
var listeners = getEventListeners(window);
if (listeners.message != null && listeners.message != undefined) {
    for (var i = 0; i < listeners.message.length; i++) {
        console.log("bufsnake %s \n"+listeners.message[i].listener.toString());
    }
}`, urlstr), &r).Do(ctx)
			return err
		}),
		chromedp.Title(title),
		fingeractions,
		chromedp.Sleep(1 * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			root, err := dom.GetDocument().WithDepth(1).Do(ctx)
			if err != nil {
				return err
			}
			html, err := dom.GetOuterHTML().WithBackendNodeID(root.BackendNodeID).Do(ctx)
			if err != nil {
				return err
			}
			*body = html
			// tab从右往左进行关闭 context deadline exceeded 就是因为这个
			err = target.CloseTarget(chromedp.FromContext(ctx).Target.TargetID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
			if err != nil {
				return errors.New(fmt.Sprintf("CloseTarget %s", err))
			}
			return nil
		}),
	}
}
