package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
)

var (
	redisUrl  string
	separator string
	keysLen   int
	mergeLen  int
	noExpire  bool
	output    string
	buildTime string
	gitHash   string
)

func init() {
	flag.StringVar(&redisUrl, "u", "redis://127.0.0.1:6379/0", "redis url, like [redis|rediss]://[:password@]host[:port][/database]")
	flag.StringVar(&separator, "s", "", "key separator, required")
	flag.IntVar(&keysLen, "sn", 100, "number of key samples")
	flag.IntVar(&mergeLen, "mn", 20, "number of merge key nodes")
	flag.BoolVar(&noExpire, "n", false, "only check no expire keys")
	flag.StringVar(&output, "o", "./", "output csv report file path")

	flag.Usage = func() {
		fmt.Printf(`redis-paser version %s, build at %s
Copyright (C) 2015-2021 by Zivn.
Web site: https://may.ltd/

redis-paser can analyze the size statistics of all keys in the redis instance and generate a csv report.

Usage: redis-paser [-u url] -s separator [-sn sample_num] [-mn merge_num] [-n] [-o ouput_dir]

Supported redis URLs are in any of these formats:
  redis://[:PASSWORD@]HOST[:PORT][/DATABASE]
  rediss://[:PASSWORD@]HOST[:PORT][/DATABASE]

Options
  -u	redis url (default: redis://127.0.0.1:6379/0)
  -s	key separator
  -sn	sample size of keys (default: 100)
  -mn	number of keys for merge key classification (default: 20)
  -n	only check keys without expiration (default: false) 
  -o	directory to save the csv report (default: "./")

`, gitHash, buildTime)
	}
}

func main() {
	// parse flag
	flag.Parse()

	if separator == "" || keysLen <= 0 || mergeLen <= 0 || output == "" {
		flag.Usage()
		return
	}

	// set max cpu core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// init paser
	paser, err := NewPaser(redisUrl, separator, keysLen, mergeLen, output)

	if err != nil {
		log.Fatalf("Fatal Error: init paser failed, redis url '%s', output '%s', %s", redisUrl, output, err)
	}

	// do parse
	err = paser.Run(noExpire)

	if err != nil {
		log.Fatalf("Fatal Error: parse item data failed, %s", err)
	}

	// save report
	err = paser.Save()

	if err != nil {
		log.Fatalf("Fatal Error: save report failed, %s", err)
	}
}
