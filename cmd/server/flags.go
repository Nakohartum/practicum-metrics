package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type JSONConfig struct {
	Address         *string `json:"address"`
	StoreInterval   *int64  `json:"store_interval"`
	FileStoragePath *string `json:"file_storage_path"`
	Restore         *bool   `json:"restore"`
	DatabaseDSN     *string `json:"database_dsn"`
	SecretKey       *string `json:"secret_key"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
	CryptoKey       *string `json:"crypto_key"`
}

// Config contains server runtime settings parsed from flags and environment.
type Config struct {
	Address         Address
	FileWork        FileWork
	DatabaseAddress DatabaseAddress
	secretKey       string `env:"SECRET_KEY"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditUrl        string `env:"AUDIT_URL"`
	CryptoKey       string `env:"CRYPTO_KEY"`
}

// DatabaseAddress stores the database connection string.
type DatabaseAddress struct {
	connectionString string `env:"DATABASE_DSN"`
}

// Address stores the server listen address.
type Address struct {
	url string `env:"ADDRESS"`
}

// String returns the address as host:port.
func (a *Address) String() string {
	return a.url
}

// Set validates and stores the address value.
func (a *Address) Set(value string) error {
	res := strings.Split(value, ":")

	if len(res) != 2 {
		return fmt.Errorf("bad address %q, want host:port", value)
	}

	a.url = value
	return nil
}

// FileWork contains file persistence settings.
type FileWork struct {
	storeInterval   int64  `env:"STORE_INTERVAL"`
	fileStoragePath string `env:"FILE_STORAGE_PATH"`
	restore         bool   `env:"RESTORE"`
}

var configData = Config{
	Address: Address{
		url: "localhost:8080",
	},
	FileWork: FileWork{
		storeInterval:   2,
		fileStoragePath: "",
		restore:         false,
	},
	DatabaseAddress: DatabaseAddress{
		connectionString: "",
	},
}

func applyJSONConfig(cfg JSONConfig) {
	if cfg.Address != nil {
		_ = configData.Address.Set(*cfg.Address)
	}
	if cfg.StoreInterval != nil {
		configData.FileWork.storeInterval = *cfg.StoreInterval
	}
	if cfg.FileStoragePath != nil {
		configData.FileWork.fileStoragePath = *cfg.FileStoragePath
	}
	if cfg.Restore != nil {
		configData.FileWork.restore = *cfg.Restore
	}
	if cfg.DatabaseDSN != nil {
		configData.DatabaseAddress.connectionString = *cfg.DatabaseDSN
	}
	if cfg.SecretKey != nil {
		configData.secretKey = *cfg.SecretKey
	}
	if cfg.AuditFile != nil {
		configData.AuditFile = *cfg.AuditFile
	}
	if cfg.AuditURL != nil {
		configData.AuditUrl = *cfg.AuditURL
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
	var jsonFilePath string
	addressFlag := configData.Address
	storeIntervalFlag := configData.FileWork.storeInterval
	fileStoragePathFlag := configData.FileWork.fileStoragePath
	restoreFlag := configData.FileWork.restore
	databaseDSNFlag := configData.DatabaseAddress.connectionString
	secretKeyFlag := configData.secretKey
	auditURLFlag := configData.AuditUrl
	auditFileFlag := configData.AuditFile
	cryptoKeyFlag := configData.CryptoKey

	flag.StringVar(&jsonFilePath, "c", "", "path to JSON config")
	flag.StringVar(&jsonFilePath, "config", "", "path to JSON config")

	flag.Var(&addressFlag, "a", "server address (host:port)")
	flag.Int64Var(&storeIntervalFlag, "i", configData.FileWork.storeInterval, "store interval in seconds")
	flag.StringVar(&fileStoragePathFlag, "f", configData.FileWork.fileStoragePath, "path to store data")
	flag.BoolVar(&restoreFlag, "r", configData.FileWork.restore, "true for restore, false for not")
	flag.StringVar(&databaseDSNFlag, "d", configData.DatabaseAddress.connectionString, "connection string for database")
	flag.StringVar(&secretKeyFlag, "k", configData.secretKey, "secret key for signing data")
	flag.StringVar(&auditURLFlag, "audit-url", configData.AuditUrl, "path to audit log url")
	flag.StringVar(&auditFileFlag, "audit-file", configData.AuditFile, "path to audit log file")
	flag.StringVar(&cryptoKeyFlag, "crypto-key", configData.CryptoKey, "key for encrypting data")
	flag.Parse()


	if err := loadJSONConfig(jsonFilePath); err != nil {
		fmt.Printf("failed to load config file %q: %v\n", jsonFilePath, err)
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			configData.Address = addressFlag
		case "i":
			configData.FileWork.storeInterval = storeIntervalFlag
		case "f":
			configData.FileWork.fileStoragePath = fileStoragePathFlag
		case "r":
			configData.FileWork.restore = restoreFlag
		case "d":
			configData.DatabaseAddress.connectionString = databaseDSNFlag
		case "k":
			configData.secretKey = secretKeyFlag
		case "audit-url":
			configData.AuditUrl = auditURLFlag
		case "audit-file":
			configData.AuditFile = auditFileFlag
		case "crypto-key":
			configData.CryptoKey = cryptoKeyFlag
		}
	})

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

	if v, ok := os.LookupEnv("SECRET_KEY"); ok && v != "" {
		configData.secretKey = v
	} else if v, ok := os.LookupEnv("KEY"); ok && v != "" {
		configData.secretKey = v
	}

	if v, ok := os.LookupEnv("AUDIT_FILE"); ok && v != "" {
		configData.AuditFile = v
	}

	if v, ok := os.LookupEnv("AUDIT_URL"); ok && v != "" {
		configData.AuditUrl = v
	}

	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		configData.CryptoKey = v
	}
}
