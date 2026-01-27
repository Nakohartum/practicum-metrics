package main

import (
	"flag"
)

var addressString string

func parseFlags() {
	flag.StringVar(&addressString, "a", ":8080", "server address")
	flag.Parse()
}