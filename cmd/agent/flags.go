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
	host string `env:"AGENT_HOST"`
	port string `env:"AGENT_PORT"`
}


func (a *Address) String() string {
	host := a.host
	if host == "" {
		host = "localhost"
	}
	port := a.port
	if port == "" {
		port = "8080"
	}
	return "http://" + host + ":" + port
}

func (a *Address) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad address %q, want host:port", value)
	}
	a.host = parts[0]
	a.port = parts[1]
	if a.port == "" {
		a.port = "8080"
	}
	return nil
}

var reportInterval int64
var pollInterval int64
var address = Address{host: "localhost", port: "8080"}

func parseFlags() {

	hostErr := env.Parse(&address)
	if hostErr != nil {
		flag.Var(&address, "a", "server address (host:port)")
	}

	reportInt, repIntErr := strconv.ParseInt(os.Getenv("REPORT_INTERVAL"), 10, 64)
	if repIntErr != nil {
		flag.Int64Var(&reportInterval, "r", 10, "report interval")
	}else{
		reportInterval = reportInt
	}


	pollInt, polIntErr := strconv.ParseInt(os.Getenv("POLL_INTERVAL"), 10, 64)
	if polIntErr != nil {
		flag.Int64Var(&pollInterval, "p", 2, "poll interval")
	}else{
		pollInterval = pollInt
	}
	flag.Parse()
}