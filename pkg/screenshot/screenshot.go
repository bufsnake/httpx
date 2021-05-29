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
	url     string
	timeout int
	path    string
}

func (c *chrome) Run() (string, error) {
	buf, err := c.runChromedp(c.url)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64, " + base64.StdEncoding.EncodeToString(buf), nil
}

func (c *chrome) runChromedp(url string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// 启动界面
		// chromedp.Flag("headless", false),
		chromedp.UserAgent(useragent.RandomUserAgent()),
		chromedp.WindowSize(1920, 720),
		chromedp.DisableGPU,
		chromedp.NoSandbox,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("incognito", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.ExecPath(c.path),
		// 隐藏滚动条
		//chromedp.Flag("hide-scrollbars", false),
	)
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(c.timeout))
	ctx, cancel = chromedp.NewContext(ctx) //chromedp.WithLogf(log.Printf),
	//chromedp.WithDebugf(log.Printf),
	//chromedp.WithErrorf(log.Printf),

	defer cancel()
	var buf []byte

	if err := chromedp.Run(ctx, fullScreenshot(url, 90, &buf)); err != nil {
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

			//width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))
			width, height := int64(1920), int64(1080)

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}
			// capture screenshot
			*res, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X: contentSize.X,
					Y: contentSize.Y,
					//Width:  contentSize.Width,
					//Height: contentSize.Height,
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
