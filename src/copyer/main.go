package main

import (
	_ "embed"

	"flag"
	"fmt"
	"log"
	"runtime"
)

var (
	sourceUrl    string
	sourcePrefix string
	targetUrl    string
	targetPrefix string

	buildTime string
	gitHash   string

	//go:embed usage.txt
	usage string
)

func init() {
	flag.StringVar(&sourceUrl, "su", "redis://127.0.0.1:6379/0", "")
	flag.StringVar(&sourcePrefix, "sp", "", "")
	flag.StringVar(&targetUrl, "tu", "redis://127.0.0.1:6379/0", "")
	flag.StringVar(&targetPrefix, "tp", "", "")

	flag.Usage = func() {
		fmt.Printf(usage, gitHash, buildTime)
	}
}

func main() {
	// parse flag
	flag.Parse()

	if sourcePrefix == "" || targetPrefix == "" {
		flag.Usage()
		return
	}

	// set max cpu core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// init copyer
	copyer, err := NewCopyer(sourceUrl, targetUrl)

	if err != nil {
		log.Fatalf("Fatal Error: invalid redis url '%s' '%s', %s", sourceUrl, targetUrl, err)
	}

	// do copy
	err = copyer.Run(sourcePrefix, targetPrefix)

	if err != nil {
		log.Fatalf("Fatal Error: copy '%s' to '%s' failed, %s", sourcePrefix, targetPrefix, err)
	}
}
