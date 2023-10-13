package main

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
	"time"
)

var flagRunAddr string
var flagReportInterval time.Duration
var flagPollInterval time.Duration

type config struct {
	Address        string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.DurationVar(&flagReportInterval, "r", 10, "report interval")
	flag.DurationVar(&flagPollInterval, "p", 2, "poll interval")
	flag.Parse()

	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Address != "" {
		flagRunAddr = cfg.Address
	}
	if cfg.ReportInterval != 0 {
		flagReportInterval = cfg.ReportInterval
	}
	if cfg.PollInterval != 0 {
		flagPollInterval = cfg.PollInterval
	}
}
