package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address Address
	FileWork FileWork
	DatabaseAddress DatabaseAddress
}

type DatabaseAddress struct {
	connectionString string `env:"DATABASE_DSN"`
}

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

type FileWork struct {
	storeInterval int64 `env:"STORE_INTERVAL"`
	fileStoragePath string `env:"FILE_STORAGE_PATH"`
	restore bool `env:"RESTORE"`
}

var configData = Config{
	Address: Address{
		url: "localhost:8080",
	},
	FileWork: FileWork{
		storeInterval: 2,
		fileStoragePath: "",
		restore: false,
	},
	DatabaseAddress: DatabaseAddress{
		connectionString: "",
	},
}

func parseFlags() {
	
	flag.Var(&configData.Address, "a", "server address (host:port)")
	flag.Int64Var(&configData.FileWork.storeInterval, "i", 2, "store interval in seconds")
	flag.StringVar(&configData.FileWork.fileStoragePath, "f", "", "path to store data")
	flag.BoolVar(&configData.FileWork.restore, "r", false, "true for restore, false for not")
	flag.StringVar(&configData.DatabaseAddress.connectionString, "d", "", "connection string for database")
	flag.Parse()

	
	if v, ok := os.LookupEnv("ADDRESS"); ok && v != "" {
		_ = configData.Address.Set(v)
	}

	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok && v != "" {
		res, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			fmt.Printf("bad store interval %q, want integer: %v\n", v, err)
		}
		configData.FileWork.storeInterval = res
	}

	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok && v != "" {
		configData.FileWork.fileStoragePath = v
	}

	if v, ok := os.LookupEnv("RESTORE"); ok && v != "" {
		res, err := strconv.ParseBool(v)
		if err != nil {
			fmt.Printf("bad restore value %q, want boolean: %v\n", v, err)
		}
		configData.FileWork.restore = res
	}

	if v, ok := os.LookupEnv("DATABASE_DSN"); ok && v != "" {
		configData.DatabaseAddress.connectionString = v
	}
}