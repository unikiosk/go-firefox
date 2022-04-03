package main

import (
	"context"
	"log"
	"time"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	ctx := context.Background()
	ui, err := gofirefox.New("https://synpse.net", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Stop()

	go func() {
		err := ui.Run(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()
	// Wait until UI window is closed

	time.Sleep(time.Second * 10)

	err = ui.Load("https://synpse.net/blog/")
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}
