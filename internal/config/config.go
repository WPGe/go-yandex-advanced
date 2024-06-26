package config

import (
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type Config struct {
	Address         string `env:"ADDRESS"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	RootDir         string `env:"ROOT_DIR"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollInterval    int    `env:"POLL_INTERVAL"`
	HashKey         string `env:"KEY"`
	RateLimit       int    `env:"RATE_LIMIT"`
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
	if config.HashKey == "" {
		config.HashKey = flags.HashKey
	}

	startDebugLogs()

	return config, nil
}

func parseServerFlags() Config {
	flagRunAddr := flag.String("a", "localhost:8080", "address and port to run server")
	flagStoreInterval := flag.Int64("i", 300, "time interval when metrics saved to file")
	flagFileStoragePath := flag.String("f", "/tmp/metrics-db.json", "filepath where the current metrics are saved")
	flagRestore := flag.Bool("r", true, "load previously saved metrics from a file at startup")
	flagDatabaseDSN := flag.String("d", "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable", "database DSN")
	flagHashKey := flag.String("k", "supersecretkey", "hash key")
	flag.Parse()

	return Config{
		Address:         *flagRunAddr,
		StoreInterval:   *flagStoreInterval,
		FileStoragePath: *flagFileStoragePath,
		Restore:         *flagRestore,
		DatabaseDSN:     *flagDatabaseDSN,
		HashKey:         *flagHashKey,
	}
}

func NewAgent() (Config, error) {
	flags := parseAgentFlags()

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
	if config.ReportInterval == 0 {
		config.ReportInterval = flags.ReportInterval
	}
	if config.PollInterval == 0 {
		config.PollInterval = flags.PollInterval
	}
	if config.HashKey == "" {
		config.HashKey = flags.HashKey
	}
	if config.RateLimit == 0 {
		config.RateLimit = flags.RateLimit
	}

	startDebugLogs()

	return config, nil
}

func parseAgentFlags() Config {
	flagRunAddr := flag.String("a", "localhost:8080", "address and port to run server")
	flagReportInterval := flag.Int("r", 10, "report interval")
	flagPollInterval := flag.Int("p", 2, "poll interval")
	flagHashKey := flag.String("k", "supersecretkey", "hash key")
	flagRateLimit := flag.Int("l", 3, "rate limit")
	flag.Parse()

	return Config{
		Address:        *flagRunAddr,
		ReportInterval: *flagReportInterval,
		PollInterval:   *flagPollInterval,
		HashKey:        *flagHashKey,
		RateLimit:      *flagRateLimit,
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
