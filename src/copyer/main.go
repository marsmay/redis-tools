package main

import (
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
	buildTime    string
	gitHash      string
)

func init() {
	flag.StringVar(&sourceUrl, "su", "redis://127.0.0.1:6379/0", "source redis url, like [redis|rediss]://[:password@]host[:port][/database]")
	flag.StringVar(&sourcePrefix, "sp", "", "source key prefix, required")
	flag.StringVar(&targetUrl, "tu", "redis://127.0.0.1:6379/0", "target redis url, like [redis|rediss]://[:password@]host[:port][/database]")
	flag.StringVar(&targetPrefix, "tp", "", "target key prefix, required")

	flag.Usage = func() {
		fmt.Printf(`redis-copyer version %s, build at %s
Copyright (C) 2015-2021 by Zivn.
Web site: https://may.ltd/

redis-copyer can copy the keys of the specified prefix from one redis instance to another redis instance.

Usage: redis-copyer [-su url] -sp prefix [-tu url] -tp prefix

Supported redis URLs are in any of these formats:
  redis://[:PASSWORD@]HOST[:PORT][/DATABASE]
  rediss://[:PASSWORD@]HOST[:PORT][/DATABASE]

Options
  -su	source redis url (default: redis://127.0.0.1:6379/0)
  -sp	source key prefix 
  -tu 	target redis url (default: redis://127.0.0.1:6379/0)
  -tp   target key prefix

`, gitHash, buildTime)
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
