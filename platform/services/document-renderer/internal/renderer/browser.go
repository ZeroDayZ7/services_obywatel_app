package renderer

import (
	"fmt"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func NewBrowser(chromeBinPath string) (*rod.Browser, error) {
	l := launcher.New().
		NoSandbox(true).
		Headless(true).
		Set("disable-dev-shm-usage", "true").
		Set("disable-gpu", "true")

	if chromeBinPath != "" {
		l = l.Bin(chromeBinPath)
	} else if path, found := launcher.LookPath(); found {
		l = l.Bin(path)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch chromium: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect browser: %w", err)
	}

	return browser, nil
}
