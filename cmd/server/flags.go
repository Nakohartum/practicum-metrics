package main

import (
	"flag"
	"strings"
)

type Address struct{
	url string
	port string
}

func (a *Address) String() string{
	return a.url + ":" + a.port
}

func (a *Address) Set(value string) error{
	res := strings.Split(value, ":")

	if res[0] == ""{
		res[0] = "localhost"
	}
	a.url = res[0]
	a.port = res[1]
	return nil
}

var address Address = Address{
	url: "localhost",
	port: "8080",
}
func parseFlags() {
	flag.Var(&address, "a", "server address")
	flag.Parse()
}