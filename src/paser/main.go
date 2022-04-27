package main

import (
	_ "embed"

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

	//go:embed usage.txt
	usage string
)

func init() {
	flag.StringVar(&redisUrl, "u", "redis://127.0.0.1:6379/0", "")
	flag.StringVar(&separator, "s", "", "")
	flag.IntVar(&keysLen, "sn", 100, "")
	flag.IntVar(&mergeLen, "mn", 20, "")
	flag.BoolVar(&noExpire, "n", false, "")
	flag.StringVar(&output, "o", "./", "")

	flag.Usage = func() {
		fmt.Printf(usage, gitHash, buildTime)
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
