package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)
type Address struct {
	host string `env:"ADDRESS"`
}


func (a *Address) String() string {
    host, port, ok := strings.Cut(a.host, ":")
    if !ok {
        host = a.host
        port = ""
    }
    if host == "" {
        host = "localhost"
    }
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

    
    if v, ok := os.LookupEnv("ADDRESS"); ok {
        _ = address.Set(v)
    }

    if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
        if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
            reportInterval = intVal
        }
    }
    if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
        if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
            pollInterval = intVal
        }
    }
}
