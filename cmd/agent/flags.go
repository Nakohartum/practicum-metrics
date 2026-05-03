package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	address        Address
	reportInterval int64  `env:"REPORT_INTERVAL"`
	pollInterval   int64  `env:"POLL_INTERVAL"`
	secretKey      string `env:"KEY"`
	rateLimit      int64  `env:"RATE_LIMIT"`
}

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

	if len(res) != 2 {
		return fmt.Errorf("bad address %q, want host:port", value)
	}

	a.host = value
	return nil
}

var configData = Config{
	address:        Address{host: "localhost:8080"},
	reportInterval: 10,
	pollInterval:   2,
}

func parseFlags() {
	flag.Var(&configData.address, "a", "server address (host:port)")
	flag.Int64Var(&configData.reportInterval, "r", 10, "report interval")
	flag.Int64Var(&configData.pollInterval, "p", 2, "poll interval")
	flag.StringVar(&configData.secretKey, "k", "", "secret key for signing data")
	flag.Int64Var(&configData.rateLimit, "l", 1024, "amount of workers")
	flag.Parse()

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		_ = configData.address.Set(v)
	}

	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			configData.reportInterval = intVal
		}
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			configData.pollInterval = intVal
		}
	}
	if v, ok := os.LookupEnv("KEY"); ok {
		configData.secretKey = v
	}
	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			configData.rateLimit = intVal
		}
	}
}
