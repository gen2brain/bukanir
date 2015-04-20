package main

import (
	"golang.org/x/mobile/app"

	_ "bukanir/go_bukanir"
	"golang.org/x/mobile/bind/java"
)

func main() {
	app.Run(app.Callbacks{Start: java.Init})
}
