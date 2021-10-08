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
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"regexp"
	"strings"
	"sync"
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

// png/icp/err
func (c *chrome) Run(url string) (string, string, string, map[string]bool, error) {
	buf, icp, title, reqs, err := c.runChromedp(url)
	if err != nil {
		return "", "", "", nil, err
	}
	return base64.StdEncoding.EncodeToString(buf), icp, title, reqs, nil
}

// Init Start CTX
func (c *chrome) InitEnv() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
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

// 获取
func (c *chrome) listen(ctx context.Context, lock *sync.Mutex, request map[string]bool, lock_ *sync.Mutex, responseId *map[network.RequestID]bool) func(ev interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			if !c.conf_.GetPath && !c.conf_.GetUrl {
				return
			}
			if !strings.HasPrefix(e.Request.URL, "http") {
				return
			}
			lock.Lock()
			request[e.Request.URL] = true
			lock.Unlock()
		case *network.EventResponseReceived:
			if !c.conf_.GetPath && !c.conf_.GetUrl {
				return
			}
			if !strings.HasPrefix(e.Response.URL, "http") {
				return
			}
			// 获取requestId
			lock_.Lock()
			(*responseId)[e.RequestID] = true
			lock_.Unlock()
		case *runtime.EventConsoleAPICalled:
			if !c.conf_.GetPath && !c.conf_.GetUrl {
				return
			}
			for i := 0; i < len(e.Args); i++ {
				var val string
				err := json.Unmarshal(e.Args[i].Value, &val)
				if err != nil {
					c.l.Error(err)
					continue
				}
				if strings.HasPrefix(val, "bufsnake") {
					split := strings.Split(val, "bufsnake ")
					if len(split) == 2 && strings.HasPrefix(split[1], "http") {
						lock.Lock()
						request[split[1]] = true
						lock.Unlock()
					}
				}
			}
		case *fetch.EventRequestPaused:
			if len(c.conf_.Headers) == 0 {
				return
			}
			// Add Headers
			// Can Not Set Host Header
			go func() {
				err := fetch.ContinueRequest(e.RequestID).WithURL(e.Request.URL).WithHeaders(c.conf_.Headers).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Target))
				if err != nil {
					c.l.Error(err)
				}
			}()
		case *page.EventJavascriptDialogOpening:
			// Disable JavaScriptDialog
			// chrome IWA 不影响截图
			t := page.HandleJavaScriptDialog(false)
			go func() {
				if err := chromedp.Run(ctx, t); err != nil {
					c.l.Error(errors.New("\nrunChromedp error: " + err.Error()))
				}
			}()
		}
	}
}

// Start Sub Tabs png/icp/title/req-url
func (c *chrome) runChromedp(url string) ([]byte, string, string, map[string]bool, error) {
	var buf []byte
	var icp string
	var title string
	requestURL := make(map[string]bool)        // 打开网页请求的URL
	jsfinder_href_src := make(map[string]bool) // JSFinder、SRC/HREF 获取的URL
	lock := sync.Mutex{}
	requestId := make(map[network.RequestID]bool)
	lock_ := sync.Mutex{}
	newContext, cancelFunc := chromedp.NewContext(c.ctx)
	defer cancelFunc()
	newContext, cancelFunc = context.WithTimeout(newContext, 30*time.Second)
	defer cancelFunc()
	chromedp.ListenTarget(newContext, c.listen(newContext, &lock, requestURL, &lock_, &requestId))
	// new tabs
	// chromedp.Run -> newTarget -> target.CreateTarget -> (p *CreateTargetParams) Do -> context canceled/context deadline exceeded
	// tabs can not auto close
	if err := chromedp.Run(newContext, c.screenshot(url, &icp, &buf, &title, &lock_, &requestId, &jsfinder_href_src)); err != nil {
		return []byte{}, "", "", requestURL, errors.New("chromedp.Run error: " + err.Error())
	}
	lock.Lock()
	defer lock.Unlock()
	for urlstr, _ := range jsfinder_href_src {
		requestURL[urlstr] = true
	}
	return buf, icp, title, requestURL, nil
}

func (c *chrome) screenshot(urlstr string, icp *string, res *[]byte, title *string, lock_ *sync.Mutex, responseId *map[network.RequestID]bool, resq *map[string]bool) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(c.conf_.Headers) > 0 {
				err := fetch.Enable().Do(ctx)
				if err != nil {
					return err
				}
				// 同样也不能修改host头
				//err := network.Enable().Do(ctx)
				//if err != nil {
				//	return err
				//}
				//m := make(map[string]interface{})
				//for i := 0; i < len(c.conf_.Headers); i++ {
				//	m[c.conf_.Headers[i].Name] = c.conf_.Headers[i].Value
				//}
				//err = network.SetExtraHTTPHeaders(m).Do(ctx)
				//if err != nil {
				//	return err
				//}
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
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			body, err := dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			if err != nil {
				return err
			}
			info := utils.ICPInfo(body)
			icp = &info
			return nil
		}),
		chromedp.Title(title),
		// 获取JSFinder
		chromedp.ActionFunc(func(ctx context.Context) error {
			if !c.conf_.GetPath && !c.conf_.GetUrl {
				return nil
			}
			for ri, _ := range *responseId {
				body, err := network.GetResponseBody(ri).Do(ctx)
				if err != nil {
					c.l.Error(err)
					continue
				}
				compile := regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|/][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:\?[^"|']{0,}|)))(?:"|')`)
				all := compile.FindAll(body, -1)
				for i := 0; i < len(all); i++ {
					url_ := strings.Trim(string(all[i]), " '\",;{}[]()`?")
					if strings.HasPrefix(url_, "http") {
						lock_.Lock()
						(*resq)[url_] = true
						lock_.Unlock()
					}
				}
			}
			return nil
		}),
		// 执行JavaScript Get Page Href/Src
		chromedp.ActionFunc(func(ctx context.Context) error {
			if !c.conf_.GetPath && !c.conf_.GetUrl {
				return nil
			}
			_, _, err := runtime.Evaluate(`
function cycle(a) {
    for (var i = 0; i < a.children.length; i++) {
        if (a.children[i].href !== undefined && a.children[i].href !== "" && a.children[i].href !== null) {
            console.log("bufsnake " + a.children[i].href);
        }
        if (a.children[i].src !== undefined && a.children[i].src !== "" && a.children[i].src !== null) {
            console.log("bufsnake " + a.children[i].src);
        }
        if (a.children[i].action !== undefined && a.children[i].action !== "" && a.children[i].action !== null) {
            console.log("bufsnake " + a.children[i].action);
        }
        cycle(a.children[i]);
    }
}
cycle(document.documentElement);
`).Do(ctx)
			return err
		}),
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// tab从右往左进行关闭 context deadline exceeded 就是因为这个
			err := target.CloseTarget(chromedp.FromContext(ctx).Target.TargetID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
			if err != nil {
				return errors.New(fmt.Sprintf("CloseTarget %s", err))
			}
			return nil
		}),
	}
}
