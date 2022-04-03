package main

import (
	"context"
	"log"
	"net/url"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	// Create UI with basic HTML passed via data URI
	ctx := context.Background()
	ui, err := gofirefox.New("data:text/html,"+url.PathEscape(`
	<html>
		<head><title>Hello</title></head>
		<body><h1>Hello, world!</h1></body>
	</html>
	`), nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Stop()

	err = ui.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}

}
