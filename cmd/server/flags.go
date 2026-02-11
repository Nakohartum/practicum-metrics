package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Address struct{
	url string  `env:"ADDRESS"`
}

func (a *Address) String() string{
	return a.url
}

func (a *Address) Set(value string) error{
	res := strings.Split(value, ":")

	if len(res) != 2{
		return fmt.Errorf("bad address %q, want host:port", value)
	}

	a.url = value
	return nil
}

var address Address = Address{
	url: "localhost:8080",
}
func parseFlags() {
	
	flag.Var(&address, "a", "server address (host:port)")
	flag.Parse()

	
	if v, ok := os.LookupEnv("ADDRESS"); ok && v != "" {
		_ = address.Set(v)
	}
}