package fitnesscert

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// HTMLToPDF renders HTML to a single-page (or multi-page) PDF using headless Chromium.
func HTMLToPDF(ctx context.Context, htmlContent string, chromiumPath string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "fitness-cert-*.html")
	if err != nil {
		return nil, fmt.Errorf("temp html: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	if _, err := tmp.WriteString(htmlContent); err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, err
	}
	defer cleanup()

	abs, err := filepath.Abs(tmpPath)
	if err != nil {
		return nil, err
	}
	fileURL := pathToFileURL(abs)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoSandbox,
		chromedp.DisableGPU,
		chromedp.Headless,
	)
	if p := strings.TrimSpace(chromiumPath); p != "" {
		opts = append([]chromedp.ExecAllocatorOption{chromedp.ExecPath(p)}, opts...)
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	runCtx, cancel := context.WithTimeout(chromeCtx, 2*time.Minute)
	defer cancel()

	var pdf []byte
	err = chromedp.Run(runCtx,
		chromedp.Navigate(fileURL),
		chromedp.ActionFunc(func(actx context.Context) error {
			var e error
			pdf, _, e = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0.35).
				WithMarginBottom(0.35).
				WithMarginLeft(0.35).
				WithMarginRight(0.35).
				Do(actx)
			return e
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("html to pdf: %w", err)
	}
	return pdf, nil
}

func pathToFileURL(abs string) string {
	slash := filepath.ToSlash(abs)
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	u := url.URL{Scheme: "file", Path: slash}
	return u.String()
}
