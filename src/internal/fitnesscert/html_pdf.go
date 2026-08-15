package fitnesscert

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// HTMLToPDF renders HTML to a single-page (or multi-page) PDF using headless Chromium.
func HTMLToPDF(ctx context.Context, htmlContent string, chromiumPath string, templateDir string) ([]byte, error) {
	tempDir := filepath.Join(strings.TrimSpace(templateDir), "tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temp dir %s: %w", tempDir, err)
	}

	tmp, err := os.CreateTemp(tempDir, "fitness-cert-*.html")
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
		chromedp.Flag("allow-file-access-from-files", true),
		// App Service: tiny /dev/shm and no D-Bus — required for stable headless Chrome.
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Env("DBUS_SESSION_BUS_ADDRESS=/dev/null"),
		chromedp.Env("GSETTINGS_BACKEND=memory"),
	)
	if p := strings.TrimSpace(chromiumPath); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
		if ld := chromeDepsLibraryPath(p); ld != "" {
			// Ensure Chrome child process sees bundled .so files (App Service has no apt root).
			if existing := strings.TrimSpace(os.Getenv("LD_LIBRARY_PATH")); existing != "" {
				ld = ld + ":" + existing
			}
			opts = append(opts, chromedp.Env("LD_LIBRARY_PATH="+ld))
		}
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	chromeCtx, cancelChrome := chromedp.NewContext(allocCtx)
	defer cancelChrome()

	runCtx, cancel := context.WithTimeout(chromeCtx, 2*time.Minute)
	defer cancel()

	var (
		reqMu          sync.Mutex
		requestURLs    = make(map[network.RequestID]string)
		missingFileURL string
	)

	chromedp.ListenTarget(runCtx, func(ev interface{}) {
		reqMu.Lock()
		defer reqMu.Unlock()

		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			requestURLs[e.RequestID] = e.Request.URL
		case *network.EventLoadingFailed:
			if !strings.Contains(e.ErrorText, "ERR_FILE_NOT_FOUND") {
				return
			}
			if u := requestURLs[e.RequestID]; u != "" {
				missingFileURL = u
				return
			}
			if e.Canceled {
				return
			}
			missingFileURL = fileURL
		}
	})

	var pdf []byte
	err = chromedp.Run(runCtx,
		network.Enable(),
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
		reqMu.Lock()
		defer reqMu.Unlock()
		if missingFileURL != "" {
			return nil, fmt.Errorf("html to pdf: %w (missing resource: %s)", err, missingFileURL)
		}
		return nil, fmt.Errorf("html to pdf: %w", err)
	}
	return pdf, nil
}

// chromeDepsLibraryPath returns LD_LIBRARY_PATH entries for chrome-linux-deps next to chrome-linux64.
// Layout: <job>/chrome-linux64/chrome and <job>/chrome-linux-deps/{lib,usr/lib}/x86_64-linux-gnu
func chromeDepsLibraryPath(chromiumPath string) string {
	chromeDir := filepath.Dir(filepath.Clean(chromiumPath))
	jobDir := filepath.Dir(chromeDir)
	depsRoot := filepath.Join(jobDir, "chrome-linux-deps")
	if st, err := os.Stat(depsRoot); err != nil || !st.IsDir() {
		return ""
	}
	var parts []string
	for _, rel := range []string{
		"lib/x86_64-linux-gnu",
		"usr/lib/x86_64-linux-gnu",
		"lib64",
	} {
		p := filepath.Join(depsRoot, rel)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ":")
}

func pathToFileURL(abs string) string {
	slash := filepath.ToSlash(abs)
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	u := url.URL{Scheme: "file", Path: slash}
	return u.String()
}
