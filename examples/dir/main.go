package main

import (
	"context"
	"log"
	"os"

	gofirefox "github.com/unikiosk/go-firefox"
)

func main() {
	ctx := context.Background()
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	ui, err := gofirefox.New(path+"/examples/dir/index.html", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ui.Stop()

	err = ui.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
