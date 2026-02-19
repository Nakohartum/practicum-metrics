package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
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

type FileWork struct {
	storeInterval int64 `env:"STORE_INTERVAL"`
	fileStoragePath string `env:"FILE_STORAGE_PATH"`
	restore bool `env:"RESTORE"`
}

var fileWork = FileWork{
	storeInterval: 2,
	fileStoragePath: "file.json",
	restore: false,
}

func parseFlags() {
	
	flag.Var(&address, "a", "server address (host:port)")
	flag.Int64Var(&fileWork.storeInterval, "i", 2, "store interval in seconds")
	flag.StringVar(&fileWork.fileStoragePath, "f", "file.json", "path to store data")
	flag.BoolVar(&fileWork.restore, "r", false, "true for restore, false for not")
	flag.Parse()

	
	if v, ok := os.LookupEnv("ADDRESS"); ok && v != "" {
		_ = address.Set(v)
	}

	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok && v != "" {
		res, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			fmt.Printf("bad store interval %q, want integer: %v\n", v, err)
		}
		fileWork.storeInterval = res
	}

	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && v != "" {
		fileWork.fileStoragePath = v
	}

	if v, ok := os.LookupEnv("RESTORE"); ok && v != "" {
		res, err := strconv.ParseBool(v)
		if err != nil {
			fmt.Printf("bad restore value %q, want boolean: %v\n", v, err)
		}
		fileWork.restore = res
	}
}