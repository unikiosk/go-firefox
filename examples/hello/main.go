package main

import (
	"log"
	"net/url"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	// Create UI with basic HTML passed via data URI
	ui, err := gofirefox.New("data:text/html,"+url.PathEscape(`
	<html>
		<head><title>Hello</title></head>
		<body><h1>Hello, world!</h1></body>
	</html>
	`), "")
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Close()
	// Wait until UI window is closed
	<-ui.Done()
}
