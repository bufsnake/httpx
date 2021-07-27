package screenshot

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/bufsnake/httpx/config"
	"github.com/bufsnake/httpx/pkg/useragent"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"strings"

	//	log2 "log"
	"time"
)

type chrome struct {
	timeout int
	ctx     context.Context
	cancel  context.CancelFunc
	conf_   config.Terminal
}

func (c *chrome) Run(url string) (string, error) {
	buf, err := c.runChromedp(url)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64, " + base64.StdEncoding.EncodeToString(buf), nil
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
	for {
		targets, err := chromedp.Targets(c.ctx)
		if err != nil {
			continue
		}
		for i := 0; i < len(targets); i++ {
			err = target.ActivateTarget(targets[i].TargetID).Do(cdp.WithExecutor(c.ctx, chromedp.FromContext(c.ctx).Browser))
			if err != nil {
				continue
			}
		}
		time.Sleep(3 * time.Second)
	}
}

// End Start CTX
func (c *chrome) Cancel() {
	defer c.cancel()
}

// Start Sub Tabs
func (c *chrome) runChromedp(url string) ([]byte, error) {
	var buf []byte
	newContext, cancelFunc := chromedp.NewContext(c.ctx)
	defer cancelFunc()
	newContext, cancelFunc = context.WithTimeout(newContext, 60*time.Second)
	defer cancelFunc()
	chromedp.ListenTarget(newContext, func(ev interface{}) {
		// JS Dialog
		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			/*
				2021/06/02 01:04:13 <- {"method":"Log.entryAdded","params":{"entry":{"source":"network","level":"error","text":"Failed to load resource: the server responded with a status of 401 (Unauthorized)","timestamp":1.6225670530905408e+12,"url":"http://42.192.77.89:9200/v1/","networkRequestId":"57902.19"}},"sessionId":"D5B25250902223AE0FD9918AE16CB241"}
				2021/06/02 01:04:13 <- {"method":"Page.javascriptDialogOpening","params":{"url":"http://42.192.77.89:9200/#/","message":"Error: Request failed with status code 401","type":"alert","hasBrowserHandler":false,"defaultPrompt":""},"sessionId":"D5B25250902223AE0FD9918AE16CB241"}
				2021/06/02 01:04:16 -> {"id":30,"sessionId":"D5B25250902223AE0FD9918AE16CB241","method":"Page.captureScreenshot","params":{"format":"png","quality":100,"clip":{"x":0,"y":0,"width":1920,"height":1080,"scale":1},"fromSurface":true,"captureBeyondViewport":true}}
			*/
			// Disable JavaScriptDialog
			t := page.HandleJavaScriptDialog(false)
			go func() {
				if err := chromedp.Run(newContext, t); err != nil {
					fmt.Println(errors.New("\nrunChromedp error: " + err.Error()))
				}
			}()
		}
		// chrome IWA
		// 不影响截图
		if _, ok := ev.(*network.EventResponseReceived); ok {
			//if ev.(*network.EventResponseReceived).Response.Headers["Www-Authenticate"] != nil {
			//	fmt.Println(ev.(*network.EventResponseReceived).Response.Headers["Www-Authenticate"])
			//}
		}
	})
	// new tabs
	// chromedp.Run -> newTarget -> target.CreateTarget -> (p *CreateTargetParams) Do -> context canceled/context deadline exceeded
	// tabs can not auto close
	if err := chromedp.Run(newContext, captureScreenshot(url, &buf)); err != nil {
		return []byte{}, errors.New("chromedp.Run error: " + err.Error())
	}
	return buf, nil
}

func captureScreenshot(urlstr string, res *[]byte) chromedp.Tasks {
	bypass_headless_detect := `(function(w, n, wn) {
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
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(bypass_headless_detect).Do(ctx)
			if err != nil {
				return errors.New(fmt.Sprintf("AddScriptToEvaluateOnNewDocument %s", err))
			}
			err = chromedp.Navigate(urlstr).Do(ctx)
			if err != nil {
				// 401 验证问题
				if !strings.Contains(err.Error(), "page load error net::ERR_INVALID_AUTH_CREDENTIALS") {
					// statuscode = 30x, close table
					// tab从右往左进行关闭 context deadline exceeded 就是因为这个
					errs := target.CloseTarget(chromedp.FromContext(ctx).Target.TargetID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
					if errs != nil {
						return errors.New(fmt.Sprintf("Navigate CloseTarget %s", err))
					}
					return err
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
			// tab从右往左进行关闭 context deadline exceeded 就是因为这个
			err := target.CloseTarget(chromedp.FromContext(ctx).Target.TargetID).Do(cdp.WithExecutor(ctx, chromedp.FromContext(ctx).Browser))
			if err != nil {
				return errors.New(fmt.Sprintf("CloseTarget %s", err))
			}
			return nil
		}),
	}
}
