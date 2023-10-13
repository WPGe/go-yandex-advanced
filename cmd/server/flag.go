package main

import (
	"flag"
	"github.com/caarlos0/env"
	"log"
)

var flagRunAddr string

type config struct {
	Address string `env:"ADDRESS"`
}

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()

	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Address != "" {
		flagRunAddr = cfg.Address
	}
}
