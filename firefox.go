package gofirefox

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

type h = map[string]interface{}

// Result is a struct for the resulting value of the JS expression or an error.
type result struct {
	Value json.RawMessage
	Err   error
}

type bindingFunc func(args []json.RawMessage) (interface{}, error)

// Msg is a struct for incoming messages (results and async events)
type msg struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type firefox struct {
	config Config
	args   []string

	sync.Mutex
	cmd      *exec.Cmd
	ws       *websocket.Conn
	id       int32
	target   string
	session  string
	window   int
	pending  map[int]chan result
	bindings map[string]bindingFunc
}

/* Firefox has a lot of configuration in profile, which is changing from release to release.
go-firefox will try to check if profile directory (either configured or provided) exists,
and attempts to use it. If profile does not exists - it will be double start
(first start created profile, kinda bootstraps it),and second start configures it and uses.
This is why we recommend some persistency of profile directory. */

// new return new firefix instance
func new(args ...string) (*firefox, error) {
	// The first two IDs are used internally during the initialization
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	args = append(args, "--profile="+config.ProfileDir)
	args = append(args, "--remote-debugging-port=0")

	fmt.Println("bootstraping firefox profile")

	c := &firefox{
		id:       2,
		config:   *config,
		args:     args,
		pending:  map[int]chan result{},
		bindings: map[string]bindingFunc{},
	}

	return c, nil
}

func (c *firefox) run(ctx context.Context) error {

	err := bootstrapFirefoxProfile(context.TODO(), &c.config)
	if err != nil {
		return err
	}
	fmt.Println("bootstraping firefox profile done")

	// Start chrome process
	c.cmd = exec.CommandContext(ctx, FirefoxExecutable(), c.args...)
	pipe, err := c.cmd.StderrPipe()
	if err != nil {
		fmt.Printf("exec.v failed %v", err)
		return err
	}
	if err := c.cmd.Start(); err != nil {
		fmt.Printf("exec.Start failed %v", err)
		return err
	}

	// Wait for websocket address to be printed to stderr
	re := regexp.MustCompile(`^DevTools listening on (ws://.*?)\r?\n$`)
	m, err := readUntilMatch(pipe, re)
	if err != nil {
		c.kill()
		fmt.Printf("readUntilMatch failed %v", err)
		return err
	}
	wsURL := m[1]

	// Open a websocket
	c.ws, err = websocket.Dial(wsURL, "", "http://127.0.0.1")
	if err != nil {
		c.kill()
		fmt.Printf("websocket.Dial failed %v", err)
		return err
	}

	// Find target and initialize session
	c.target, err = c.findTarget()
	if err != nil {
		c.kill()
		fmt.Printf("c.findTarget failed %v", err)
		return err
	}

	c.session, err = c.startSession(c.target)
	if err != nil {
		c.kill()
		fmt.Printf("startSession failed %v", err)
		return err
	}

	go c.readLoop()
	for method, args := range map[string]h{
		"Page.enable":          nil,
		"Target.setAutoAttach": {"autoAttach": true, "waitForDebuggerOnStart": false},
		"Network.enable":       nil,
		"Runtime.enable":       nil,
		"Security.enable":      nil,
		"Performance.enable":   nil,
		"Log.enable":           nil,
	} {
		if _, err := c.send(method, args); err != nil {
			c.kill()
			c.cmd.Wait()
			fmt.Printf("send failed %v", err)
			return err
		}
	}

	if !contains(c.args, "--headless") {
		win, err := c.getWindowForTarget(c.target)
		if err != nil {
			c.kill()
			fmt.Printf("getWindowForTarget failed %v", err)
			return err
		}
		c.window = win.WindowID
	}

	return nil
}

func (c *firefox) waitForReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			if c.ws != nil {
				return nil
			}
		}
	}

}

func (c *firefox) findTarget() (string, error) {
	err := websocket.JSON.Send(c.ws, h{
		"id": 0, "method": "Target.setDiscoverTargets", "params": h{"discover": true},
	})
	if err != nil {
		return "", err
	}
	for {
		m := msg{}
		if err = websocket.JSON.Receive(c.ws, &m); err != nil {
			return "", err
		} else if m.Method == "Target.targetCreated" {
			target := struct {
				TargetInfo struct {
					Type string `json:"type"`
					ID   string `json:"targetId"`
				} `json:"targetInfo"`
			}{}
			if err := json.Unmarshal(m.Params, &target); err != nil {
				return "", err
			} else if target.TargetInfo.Type == "page" {
				return target.TargetInfo.ID, nil
			}
		}
	}
}

func bootstrapFirefoxProfile(ctx context.Context, c *Config) error {
	profileExtensionsDirPath := filepath.Join(c.ProfileDir, "extensions")
	err := os.MkdirAll(profileExtensionsDirPath, 0700)
	if err != nil {
		return fmt.Errorf("cannot create extensions directory: %s", err)
	}

	if err = func() error {
		// create HTTP client
		tr := &http.Transport{}
		defer tr.CloseIdleConnections()
		client := &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		}

		// download user.js file
		userJsPath := filepath.Join(c.ProfileDir, "user.js")
		if c.ProfileLocationURL != "" {
			log.Printf("downloading user.js %s --> %s", c.ProfileDir, userJsPath)
			err = downloadFile(ctx, client, c.ProfileLocationURL, userJsPath)
			if err != nil {
				return fmt.Errorf("failed to download user.js: %s", err)
			}
		}

		// append/modify extra preferences to user.js via our script.
		// function will update inplace.
		err := configureDevTools(userJsPath)
		if err != nil {
			return fmt.Errorf("failed to configure %s - error: %s", userJsPath, err)
		}

		return nil
	}(); err != nil {
		return err
	}
	return nil
}

func (c *firefox) startSession(target string) (string, error) {
	err := websocket.JSON.Send(c.ws, h{
		"id": 1, "method": "Target.attachToTarget", "params": h{"targetId": target},
	})
	if err != nil {
		return "", err
	}
	for {
		m := msg{}
		if err = websocket.JSON.Receive(c.ws, &m); err != nil {
			return "", err
		} else if m.ID == 1 {
			if m.Error != nil {
				return "", errors.New("Target error: " + string(m.Error))
			}
			session := struct {
				ID string `json:"sessionId"`
			}{}
			if err := json.Unmarshal(m.Result, &session); err != nil {
				return "", err
			}
			return session.ID, nil
		}
	}
}

// values for devTools config:
// user_pref("devtools.chrome.enabled", true);
// user_pref("devtools.debugger.prompt-connection", false);
// user_pref("devtools.debugger.remote-enabled", true,);

// configureDevTools will append the devtools config to the profile
func configureDevTools(prefFile string) error {
	f, err := os.Open(prefFile)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// map[matcher-pattern]replacement
	mods := map[string]string{
		"devtools.chrome.enabled":             "user_pref(\"devtools.chrome.enabled\", true);",
		"devtools.debugger.prompt-connection": "user_pref(\"devtools.debugger.prompt-connection\", false);",
		"devtools.debugger.remote-enabled":    "user_pref(\"devtools.debugger.remote-enabled\", true);",
	}

	for matcher, replacement := range mods {
		var exists bool
		for idx, line := range lines {
			if strings.Contains(line, matcher) {
				lines[idx] = replacement
				exists = true
			}
		}
		if !exists {
			lines = append(lines, replacement)
		}
	}

	return os.WriteFile(prefFile, []byte(strings.Join(lines, "\n")), 0644)
}

// WindowState defines the state of the Chrome window, possible values are
// "normal", "maximized", "minimized" and "fullscreen".
type WindowState string

const (
	// WindowStateNormal defines a normal state of the browser window
	WindowStateNormal WindowState = "normal"
	// WindowStateMaximized defines a maximized state of the browser window
	WindowStateMaximized WindowState = "maximized"
	// WindowStateMinimized defines a minimized state of the browser window
	WindowStateMinimized WindowState = "minimized"
	// WindowStateFullscreen defines a fullscreen state of the browser window
	WindowStateFullscreen WindowState = "fullscreen"
)

// Bounds defines settable window properties.
type Bounds struct {
	Left        int         `json:"left"`
	Top         int         `json:"top"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`
	WindowState WindowState `json:"windowState"`
}

type windowTargetMessage struct {
	WindowID int    `json:"windowId"`
	Bounds   Bounds `json:"bounds"`
}

func (c *firefox) getWindowForTarget(target string) (windowTargetMessage, error) {
	var m windowTargetMessage
	msg, err := c.send("Browser.getWindowForTarget", h{"targetId": target})
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(msg, &m)
	return m, err
}

type targetMessageTemplate struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params struct {
		Name    string `json:"name"`
		Payload string `json:"payload"`
		ID      int    `json:"executionContextId"`
		Args    []struct {
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		} `json:"args"`
	} `json:"params"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

type targetMessage struct {
	targetMessageTemplate
	Result struct {
		Result struct {
			Type        string          `json:"type"`
			Subtype     string          `json:"subtype"`
			Description string          `json:"description"`
			Value       json.RawMessage `json:"value"`
			ObjectID    string          `json:"objectId"`
		} `json:"result"`
		Exception struct {
			Exception struct {
				Value json.RawMessage `json:"value"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	} `json:"result"`
}

func (c *firefox) readLoop() {
	for {
		m := msg{}
		if err := websocket.JSON.Receive(c.ws, &m); err != nil {
			return
		}

		if m.Method == "Target.receivedMessageFromTarget" {
			params := struct {
				SessionID string `json:"sessionId"`
				Message   string `json:"message"`
			}{}
			err := json.Unmarshal(m.Params, &params)
			if params.SessionID != c.session {
				continue
			}
			if err != nil {
				log.Println(err)
			}

			res := targetMessage{}
			err = json.Unmarshal([]byte(params.Message), &res)
			if err != nil {
				log.Println(err)
			}

			if res.ID == 0 && res.Method == "Runtime.consoleAPICalled" || res.Method == "Runtime.exceptionThrown" {
				log.Println(params.Message)
			} else if res.ID == 0 && res.Method == "Runtime.bindingCalled" {
				payload := struct {
					Name string            `json:"name"`
					Seq  int               `json:"seq"`
					Args []json.RawMessage `json:"args"`
				}{}
				json.Unmarshal([]byte(res.Params.Payload), &payload)

				c.Lock()
				binding, ok := c.bindings[res.Params.Name]
				c.Unlock()
				if ok {
					jsString := func(v interface{}) string { b, _ := json.Marshal(v); return string(b) }
					go func() {
						result, error := "", `""`
						if r, err := binding(payload.Args); err != nil {
							error = jsString(err.Error())
						} else if b, err := json.Marshal(r); err != nil {
							error = jsString(err.Error())
						} else {
							result = string(b)
						}
						expr := fmt.Sprintf(`
							if (%[4]s) {
								window['%[1]s']['errors'].get(%[2]d)(%[4]s);
							} else {
								window['%[1]s']['callbacks'].get(%[2f]d)(%[3]s);
							}
							window['%[1]s']['callbacks'].delete(%[2]d);
							window['%[1]s']['errors'].delete(%[2]d);
							`, payload.Name, payload.Seq, result, error)
						c.send("Runtime.evaluate", h{"expression": expr, "contextId": res.Params.ID})
					}()
				}
				continue
			}

			c.Lock()
			resc, ok := c.pending[res.ID]
			delete(c.pending, res.ID)
			c.Unlock()

			if !ok {
				continue
			}

			if res.Error.Message != "" {
				resc <- result{Err: errors.New(res.Error.Message)}
			} else if res.Result.Exception.Exception.Value != nil {
				resc <- result{Err: errors.New(string(res.Result.Exception.Exception.Value))}
			} else if res.Result.Result.Type == "object" && res.Result.Result.Subtype == "error" {
				resc <- result{Err: errors.New(res.Result.Result.Description)}
			} else if res.Result.Result.Type != "" {
				resc <- result{Value: res.Result.Result.Value}
			} else {
				res := targetMessageTemplate{}
				json.Unmarshal([]byte(params.Message), &res)
				resc <- result{Value: res.Result}
			}
		} else if m.Method == "Target.targetDestroyed" {
			params := struct {
				TargetID string `json:"targetId"`
			}{}
			json.Unmarshal(m.Params, &params)
			if params.TargetID == c.target {
				c.kill()
				return
			}
		}
	}
}

func (c *firefox) send(method string, params h) (json.RawMessage, error) {
	id := atomic.AddInt32(&c.id, 1)
	b, err := json.Marshal(h{"id": int(id), "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	resc := make(chan result)
	c.Lock()
	c.pending[int(id)] = resc
	c.Unlock()

	if err := websocket.JSON.Send(c.ws, h{
		"id":     int(id),
		"method": "Target.sendMessageToTarget",
		"params": h{"message": string(b), "sessionId": c.session},
	}); err != nil {
		return nil, err
	}
	res := <-resc

	return res.Value, res.Err
}

func (c *firefox) load(url string) error {
	_, err := c.send("Page.navigate", h{"url": url})
	return err
}

func (c *firefox) kill() error {
	if c.ws != nil {
		if err := c.ws.Close(); err != nil {
			return err
		}
	}
	// TODO: cancel all pending requests
	if state := c.cmd.ProcessState; state == nil || !state.Exited() {
		return c.cmd.Process.Kill()
	}
	return nil
}

func readUntilMatch(r io.ReadCloser, re *regexp.Regexp) ([]string, error) {
	br := bufio.NewReader(r)
	defer r.Close()
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected EOF. DevTool not found")
			}
			return nil, err
		} else if m := re.FindStringSubmatch(line); m != nil {
			fmt.Println(line)
			go io.Copy(ioutil.Discard, br)
			return m, nil
		}
	}
}

func contains(arr []string, x string) bool {
	for _, n := range arr {
		if x == n {
			return true
		}
	}
	return false
}
