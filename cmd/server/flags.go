package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Address struct{
	url string  `env:"SERVER_URL"`
	port string `env:"SERVER_PORT"`
}

func (a *Address) String() string{
	return a.url + ":" + a.port
}

func (a *Address) Set(value string) error{
	res := strings.Split(value, ":")

	if len(res) != 2{
		return fmt.Errorf("bad address %q, want host:port", value)
	}

	a.url = res[0]
	if res[1] == "" {
		a.port = "8080"
	} else{
		a.port = res[1]
	}
	return nil
}

var address Address = Address{
	url: "",
	port: "8080",
}
func parseFlags() {

	hostErr := env.Parse(&address)
	if hostErr != nil {
		flag.Var(&address, "a", "server address")
		flag.Parse()
	}
	
}