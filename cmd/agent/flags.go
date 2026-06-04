package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config contains agent runtime settings parsed from flags and environment.

type JSONConfig struct {
	Address        *string `json:"address"`
	ReportInterval *int64  `json:"report_interval"`
	PollInterval   *int64  `json:"poll_interval"`
	SecretKey      *string `json:"secret_key"`
	RateLimit      *int64  `json:"rate_limit"`
	CryptoKey      *string `json:"crypto_key"`
}

type Config struct {
	address        Address
	reportInterval int64  `env:"REPORT_INTERVAL"`
	pollInterval   int64  `env:"POLL_INTERVAL"`
	secretKey      string `env:"SECRET_KEY"`
	rateLimit      int64  `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

// Address stores the metrics server address.
type Address struct {
	host string `env:"ADDRESS"`
}

// String returns the address as an HTTP URL.
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

// Set validates and stores the address value.
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
	rateLimit:      1024,
}

func applyJSONConfig(cfg JSONConfig) {
	if cfg.Address != nil {
		_ = configData.address.Set(*cfg.Address)
	}
	if cfg.ReportInterval != nil {
		configData.reportInterval = *cfg.ReportInterval
	}
	if cfg.PollInterval != nil {
		configData.pollInterval = *cfg.PollInterval
	}
	if cfg.SecretKey != nil {
		configData.secretKey = *cfg.SecretKey
	}
	if cfg.RateLimit != nil {
		configData.rateLimit = *cfg.RateLimit
	}
	if cfg.CryptoKey != nil {
		configData.CryptoKey = *cfg.CryptoKey
	}
}

func loadJSONConfig(path string) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg JSONConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	applyJSONConfig(cfg)
	return nil
}

func parseFlags() {
	var configPath string
	addressFlag := configData.address
	reportIntervalFlag := configData.reportInterval
	pollIntervalFlag := configData.pollInterval
	secretKeyFlag := configData.secretKey
	rateLimitFlag := configData.rateLimit
	cryptoKeyFlag := configData.CryptoKey

	flag.StringVar(&configPath, "c", "", "path to JSON config")
	flag.StringVar(&configPath, "config", "", "path to JSON config")

	flag.Var(&addressFlag, "a", "server address (host:port)")
	flag.Int64Var(&reportIntervalFlag, "r", configData.reportInterval, "report interval")
	flag.Int64Var(&pollIntervalFlag, "p", configData.pollInterval, "poll interval")
	flag.StringVar(&secretKeyFlag, "k", configData.secretKey, "secret key for signing data")
	flag.Int64Var(&rateLimitFlag, "l", configData.rateLimit, "amount of workers")
	flag.StringVar(&cryptoKeyFlag, "crypto-key", configData.CryptoKey, "key for encrypting data")
	flag.Parse()

	if err := loadJSONConfig(configPath); err != nil {
		fmt.Printf("failed to load config file %q: %v\n", configPath, err)
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			configData.address = addressFlag
		case "r":
			configData.reportInterval = reportIntervalFlag
		case "p":
			configData.pollInterval = pollIntervalFlag
		case "k":
			configData.secretKey = secretKeyFlag
		case "l":
			configData.rateLimit = rateLimitFlag
		case "crypto-key":
			configData.CryptoKey = cryptoKeyFlag
		}
	})

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
	if v, ok := os.LookupEnv("SECRET_KEY"); ok {
		configData.secretKey = v
	}
	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
			configData.rateLimit = intVal
		}
	}
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		configData.CryptoKey = v
	}
}
