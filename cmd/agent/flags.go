package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
)
type Address struct {
	host string `env:"ADDRESS"`
}


func (a *Address) String() string {
	res := strings.Split(a.host, ":")
	host := res[0]
	if host == "" {
		host = "localhost"
	}
	port := res[1]
	if port == "" {
		port = "8080"
	}
	return "http://" + host + ":" + port
}

func (a *Address) Set(value string) error {
	res := strings.Split(value, ":")

	if len(res) != 2{
		return fmt.Errorf("bad address %q, want host:port", value)
	}

	a.host = value
	return nil
}

var reportInterval int64
var pollInterval int64
var address = Address{host: "localhost:8080"}

func parseFlags() {
	flag.Var(&address, "a", "server address (host:port)")
	flag.Int64Var(&reportInterval, "r", 10, "report interval")
	flag.Int64Var(&pollInterval, "p", 2, "poll interval")
	flag.Parse()

	env.Parse(&address)
	if value, exists := os.LookupEnv("REPORT_INTERVAL"); exists{
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err == nil{
			reportInterval = intVal
		}
		
	}
	if value, exists := os.LookupEnv("POLL_INTERVAL"); exists{
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err == nil{
			pollInterval = intVal
		}
	}
}