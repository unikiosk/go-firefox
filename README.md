# Go-Firefox

<div>
<img align="left" src="https://github.com/unikiosk/go-firefox/raw/main/go-firefox.png" alt="Go-Firefox" width="128px" height="128px" />
<br/>
<p>
	Firefox manager as a code. Enables to run Firefox and manage firefox from code.
	</br>
	!Note: This is very early stage of development. Breaking changes are expected.
</p>
<br/>
</div>


## Features

* Pure go with simple api
* Almost no dependencies

Also, limitations by design:

* Requires Firefox to be installed.
* No controller over passed in code (bindings, data)

If you want to have more control of the browser window - consider using
[webview](https://github.com/zserge/webview) library with a similar API, so
migration would be smooth.

## Example

```go
	// Create UI with basic HTML passed via data URI
	ui, err := gofirefox.New("data:text/html,"+url.PathEscape(`
	<html>
		<head><title>Hello</title></head>
		<body><h1>Hello, world!</h1></body>
	</html>
	`), nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Close()
	// Wait until UI window is closed
	<-ui.Done()
```

## Hello World

Here are the steps to run the hello world example.

```
cd examples/hello
go get
go run ./
```

## How it works

Under the hood go-firefox uses [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/) to instrument on a Firefox instance. First go-Firefox tries to locate your installed Firefox, starts a remote debugging instance binding to an ephemeral port and reads from `stderr` for the actual WebSocket endpoint. Then golang code opens a new client connection to the WebSocket server, and instruments Firefox by sending JSON messages of Chrome DevTools Protocol methods via WebSocket. 

## Configuration

`GOFIREFOX_BIN` - override firefox location

`GOFIREFOX_PROFILE_DIR` - override firefox profile location

`GOFIREFOX_DEVTOOLS_PORT` - override firefox devtools port

`GOFIREFOX_PROFILE_LOCATION` - override firefox profile location to download from.


## Inspiration

Project inspired by multiple projects:

https://github.com/GoogleChromeLabs/carlo/
https://github.com/puppeteer/puppeteer
https://github.com/zserge/lorca 
https://github.com/webview/webview