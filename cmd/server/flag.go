package main

import (
	"flag"
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"log"
	"os"
)

var flagRunAddr string
var flagStoreInterval int64
var flagFileStoragePath string
var flagRestore bool

var rootDir string

type config struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	RootDir         string `env:"ROOT_DIR"`
}

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Int64Var(&flagStoreInterval, "i", 300, "time interval when metrics saved to file")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "filepath where the current metrics are saved")
	flag.BoolVar(&flagRestore, "r", true, "load previously saved metrics from a file at startup")
	flag.Parse()

	// Открываем файл для записи логов
	file, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Unable to open log file:", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}(file)

	// Настройка вывода в файл
	log.SetOutput(file)

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	var cfg config
	err = env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Address != "" {
		flagRunAddr = cfg.Address
	}
	if cfg.StoreInterval != 0 {
		flagStoreInterval = cfg.StoreInterval
	}
	if cfg.FileStoragePath != "" {
		flagFileStoragePath = cfg.FileStoragePath
	}
	if cfg.RootDir != "" {
		rootDir = cfg.RootDir
	}
}
