package main

import (
	"log"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	ui, err := gofirefox.New("https://synpse.net", "")
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Close()
	// Wait until UI window is closed
	<-ui.Done()
}
