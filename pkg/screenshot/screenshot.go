package screenshot

import (
	"context"
	"encoding/base64"
	"github.com/bufsnake/httpx/pkg/useragent"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"time"
)

type chrome struct {
	timeout int
	path    string
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *chrome) Run(url string) (string, error) {
	buf, err := c.runChromedp(url)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64, " + base64.StdEncoding.EncodeToString(buf), nil
}

// 初始化 母CTX
func (c *chrome) InitEnv() error {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(useragent.RandomUserAgent()),
		chromedp.WindowSize(1920, 720),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("incognito", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.ExecPath(c.path),
	)
	c.ctx, c.cancel = chromedp.NewExecAllocator(context.Background(), opts...)
	// c.ctx, c.cancel = context.WithTimeout(c.ctx, time.Second*time.Duration(c.timeout))
	c.ctx, c.cancel = chromedp.NewContext(c.ctx)
	err := chromedp.Run(c.ctx, chromedp.Tasks{})
	if err != nil {
		return err
	}
	return nil
}

// 结束母CTX
func (c *chrome) Cancel() {
	defer c.cancel()
}

func (c *chrome) runChromedp(url string) ([]byte, error) {
	var buf []byte
	newContext, cancelFunc := chromedp.NewContext(c.ctx)
    newContext, cancelFunc = context.WithTimeout(newContext, time.Second*time.Duration(c.timeout))
	defer cancelFunc()
	if err := chromedp.Run(newContext, fullScreenshot(url, 90, &buf)); err != nil {
		return []byte{}, err
	}
	return buf, nil
}

func fullScreenshot(urlstr string, quality int64, res *[]byte) chromedp.Tasks {
	script := `(function(w, n, wn) {
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

	// ignore alert
	script += `window.alert = function() {};`
	script += `window.confirm = function() {};`
	script += `window.prompt = function() {};`

	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			_, err = page.AddScriptToEvaluateOnNewDocument(script).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
		chromedp.EmulateViewport(1920, 1080, chromedp.EmulateScale(2)),
		chromedp.Navigate(urlstr),
		chromedp.Sleep(time.Duration(2) * time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 卡在 Page.javascriptDialogOpening
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
					t := page.HandleJavaScriptDialog(false)
					go func() {
						if err := chromedp.Run(ctx, t); err != nil {
						}
					}()
				}
			})
			_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}
			width, height := int64(1920), int64(1080)
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}
			*res, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  1920,
					Height: 1080,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
}
