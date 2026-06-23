package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains agent runtime settings parsed from flags and environment.

type JSONConfig struct {
	Address        *string `json:"address"`
	GRPCAddress    *string `json:"grpc_address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	SecretKey      *string `json:"secret_key"`
	RateLimit      *int64  `json:"rate_limit"`
	CryptoKey      *string `json:"crypto_key"`
}

type Config struct {
	address        Address
	grpcAddress    string `env:"GRPC_ADDRESS"`
	reportInterval time.Duration
	pollInterval   time.Duration
	secretKey      string `env:"SECRET_KEY"`
	rateLimit      int64  `env:"RATE_LIMIT"`
	cryptoKey      string `env:"CRYPTO_KEY"`
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
	reportInterval: 10 * time.Second,
	pollInterval:   2 * time.Second,
	rateLimit:      1024,
}

func applyJSONConfig(cfg JSONConfig) error {
	if cfg.Address != nil {
		if err := configData.address.Set(*cfg.Address); err != nil {
			return fmt.Errorf("set address: %w", err)
		}
	}
	if cfg.GRPCAddress != nil {
		configData.grpcAddress = *cfg.GRPCAddress
	}

	if cfg.ReportInterval != nil {
		duration, err := time.ParseDuration(*cfg.ReportInterval)
		if err != nil {
			return fmt.Errorf("parse report_interval: %w", err)
		}
		configData.reportInterval = duration
	}
	if cfg.PollInterval != nil {
		duration, err := time.ParseDuration(*cfg.PollInterval)
		if err != nil {
			return fmt.Errorf("parse poll_interval: %w", err)
		}
		configData.pollInterval = duration
	}
	if cfg.SecretKey != nil {
		configData.secretKey = *cfg.SecretKey
	}
	if cfg.RateLimit != nil {
		configData.rateLimit = *cfg.RateLimit
	}
	if cfg.CryptoKey != nil {
		configData.cryptoKey = *cfg.CryptoKey
	}
	return nil
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

	return applyJSONConfig(cfg)
}

func parseFlags() error {
	var configPath string
	addressFlag := configData.address
	reportIntervalFlag := int64(configData.reportInterval / time.Second)
	pollIntervalFlag := int64(configData.pollInterval / time.Second)
	secretKeyFlag := configData.secretKey
	rateLimitFlag := configData.rateLimit
	cryptoKeyFlag := configData.cryptoKey
	grpcAddressFlag := configData.grpcAddress

	flag.StringVar(&configPath, "c", "", "path to JSON config")
	flag.StringVar(&configPath, "config", "", "path to JSON config")

	flag.Var(&addressFlag, "a", "server address (host:port)")
	flag.Int64Var(&reportIntervalFlag, "r", reportIntervalFlag, "report interval in seconds")
	flag.Int64Var(&pollIntervalFlag, "p", pollIntervalFlag, "poll interval in seconds")
	flag.StringVar(&secretKeyFlag, "k", configData.secretKey, "secret key for signing data")
	flag.Int64Var(&rateLimitFlag, "l", configData.rateLimit, "amount of workers")
	flag.StringVar(&cryptoKeyFlag, "crypto-key", configData.cryptoKey, "key for encrypting data")
	flag.StringVar(&grpcAddressFlag, "g", configData.grpcAddress, "gRPC server address (host:port)")
	flag.Parse()

	if err := loadJSONConfig(configPath); err != nil {
		return fmt.Errorf("load config file %q: %w", configPath, err)
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			configData.address = addressFlag
		case "r":
			configData.reportInterval = time.Duration(reportIntervalFlag) * time.Second
		case "p":
			configData.pollInterval = time.Duration(pollIntervalFlag) * time.Second
		case "k":
			configData.secretKey = secretKeyFlag
		case "l":
			configData.rateLimit = rateLimitFlag
		case "crypto-key":
			configData.cryptoKey = cryptoKeyFlag
		case "g":
			configData.grpcAddress = grpcAddressFlag
		}
	})

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		if err := configData.address.Set(v); err != nil {
			return fmt.Errorf("set ADDRESS: %w", err)
		}
	}

	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("parse REPORT_INTERVAL: %w", err)
		}
		configData.reportInterval = time.Duration(intVal) * time.Second
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("parse POLL_INTERVAL: %w", err)
		}
		configData.pollInterval = time.Duration(intVal) * time.Second
	}
	if v, ok := os.LookupEnv("SECRET_KEY"); ok {
		configData.secretKey = v
	}
	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("parse RATE_LIMIT: %w", err)
		}
		configData.rateLimit = intVal
	}
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		configData.cryptoKey = v
	}
	if v, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		configData.grpcAddress = v
	}
	return nil
}
