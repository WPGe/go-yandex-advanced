package config

import (
	"flag"
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	RootDir         string `env:"ROOT_DIR"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func NewServer() (Config, error) {
	flags := parseServerFlags()

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	config := Config{}
	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}

	if config.Address == "" {
		config.Address = flags.Address
	}
	if config.StoreInterval == 0 {
		config.StoreInterval = flags.StoreInterval
	}
	if config.FileStoragePath == "" {
		config.FileStoragePath = flags.FileStoragePath
	}
	if config.RootDir == "" {
		config.RootDir = flags.RootDir
	}
	if config.DatabaseDSN == "" {
		config.DatabaseDSN = flags.DatabaseDSN
	}

	startDebugLogs()

	return config, nil
}

func parseServerFlags() Config {
	flagRunAddr := flag.String("a", "localhost:8080", "address and port to run server")
	flagStoreInterval := flag.Int64("i", 300, "time interval when metrics saved to file")
	flagFileStoragePath := flag.String("f", "/tmp/metrics-db.json", "filepath where the current metrics are saved")
	flagRestore := flag.Bool("r", true, "load previously saved metrics from a file at startup")
	flagDatabaseDSN := flag.String("d", "", "database DSN")
	flag.Parse()

	return Config{
		Address:         *flagRunAddr,
		StoreInterval:   *flagStoreInterval,
		FileStoragePath: *flagFileStoragePath,
		Restore:         *flagRestore,
		DatabaseDSN:     *flagDatabaseDSN,
	}
}

func startDebugLogs() {
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
