package gofirefox

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// UI interface allows talking to the HTML5 UI from Go.
type UI interface {
	Load(url string) error

	Run(ctx context.Context) error
	Stop() error
}

type ui struct {
	firefox *firefox
	done    chan struct{}
}

var defaultArgs = []string{}

// New returns a new HTML5 UI for the given URL, user profile directory, window
// size and other options passed to the browser engine. If URL is an empty
// string - a blank page is displayed. If user profile directory is an empty
// string - a temporary directory is created and it will be removed on
// ui.Stop().
// There are 3 execution modes:
// 1. url only is provided - run firefox with the url
// 2. url is provided with prefix "data:" - run firefox with the url encoded content data:
// 3. url is directory with index.html as postfix - serve directory as file://
func New(url string, customArgs, userPreferences []string) (UI, error) {
	// there is 3 execution modes:
	// 1. url only is provided - run firefox with the url
	// 2. url is provided with prefix "data:" - run firefox with the url (same code behaviour as 1)
	// 3. url and dir is provided - service directory and open file provided in url
	if url == "" {
		url = "data:text/html,<html>Hello from Unikiosk!</html>"
	}

	// split for parsing
	urlParts := strings.Split(url, "/")
	postfix := urlParts[len(urlParts)-1]

	if strings.Contains(postfix, ".html") || strings.Contains(postfix, ".htm") || strings.Contains(postfix, ".php") {
		_, err := os.Stat(url)
		if err != nil {
			return nil, err
		}
		url = "file://" + url
	}

	args := customArgs
	args = append(args, fmt.Sprintf("--new-window=%s", url))
	args = append(args, "--kiosk")

	firefox, err := new(args, userPreferences)
	if err != nil {
		return nil, err
	}

	return &ui{firefox: firefox}, nil
}

func (u *ui) Stop() error {
	// ignore err, as the chrome process might be already dead, when user close the window.
	err := u.firefox.stop()
	if err != nil {
		return err
	}
	<-u.done
	return nil
}

func (u *ui) Load(url string) error {
	return u.firefox.load(url)
}

func (u *ui) Run(ctx context.Context) error {
	return u.firefox.run(ctx)
}
