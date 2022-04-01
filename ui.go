package gofirefox

import (
	"context"
	"fmt"
	"os"
)

// UI interface allows talking to the HTML5 UI from Go.
type UI interface {
	Load(url string) error
	Done() <-chan struct{}

	Run(ctx context.Context) error
	Close() error
}

type ui struct {
	firefox *firefox
	done    chan struct{}
	tmpDir  string
}

var defaultArgs = []string{}

// New returns a new HTML5 UI for the given URL, user profile directory, window
// size and other options passed to the browser engine. If URL is an empty
// string - a blank page is displayed. If user profile directory is an empty
// string - a temporary directory is created and it will be removed on
// ui.Close(). You might want to use "--headless" custom CLI argument to test
// your UI code.
func New(url, dir string, customArgs ...string) (UI, error) {
	if url == "" {
		url = "data:text/html,<html></html>"
	}
	tmpDir := ""
	if dir == "" {
		name, err := os.MkdirTemp("", "gofirefox")
		if err != nil {
			return nil, err
		}
		dir, tmpDir = name, name
	}
	args := append(defaultArgs, fmt.Sprintf("--new-window=%s", url))
	//args = append(args, "--kiosk")

	args = append(args, customArgs...)

	firefox, err := new(args...)
	done := make(chan struct{})
	if err != nil {
		return nil, err
	}

	go func() {
		firefox.run(context.TODO())
		close(done)
	}()

	err = firefox.waitForReady(context.TODO())
	if err != nil {
		return nil, err
	}

	return &ui{firefox: firefox, done: done, tmpDir: tmpDir}, nil
}

func (u *ui) Done() <-chan struct{} {
	return u.done
}

func (u *ui) Close() error {
	// ignore err, as the chrome process might be already dead, when user close the window.
	u.firefox.kill()
	<-u.done
	if u.tmpDir != "" {
		if err := os.RemoveAll(u.tmpDir); err != nil {
			return err
		}
	}
	return nil
}

func (u *ui) Load(url string) error { return u.firefox.load(url) }

func (u *ui) Run(ctx context.Context) error {
	return u.firefox.run(ctx)
}
