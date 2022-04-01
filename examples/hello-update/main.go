package main

import (
	"log"
	"time"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	ui, err := gofirefox.New("https://synpse.net", "")
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Close()
	// Wait until UI window is closed

	time.Sleep(time.Second * 10)

	ui.Load("https://synpse.net/blog/")

	<-ui.Done()
}
