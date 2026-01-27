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

var reportInterval int64
var pollInterval int64
var address Address = Address{
	url: "localhost",
	port: "8080",
}


func parseFlags() {
	flag.Var(&address, "a", "server address")
	flag.Int64Var(&reportInterval, "r", 10, "report interval - interval used to send data to the server")
	flag.Int64Var(&pollInterval, "p", 2, "poll interval - interval used to update agent's data")
	flag.Parse()
}