package main

import (
	"context"
	"log"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	ctx := context.Background()
	ui, err := gofirefox.New("https://synpse.net", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Stop()

	err = ui.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
