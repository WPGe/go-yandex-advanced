package main

import "flag"

var flagRunAddr string
var flagReportInterval int
var flagPollInterval int

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval")
	flag.Parse()
}
