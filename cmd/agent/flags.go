package main

import "flag"

var addressString string
var reportInterval int64
var pollInterval int64

func parseFlags() {
	flag.StringVar(&addressString, "a", ":8080", "server address")
	flag.Int64Var(&reportInterval, "r", 10, "report interval - interval used to send data to the server")
	flag.Int64Var(&pollInterval, "p", 2, "poll interval - interval used to update agent's data")
	flag.Parse()
}