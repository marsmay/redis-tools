package main

import (
	_ "embed"

	"flag"
	"fmt"
	"log"
	"runtime"
)

var (
	redisUrl    string
	separator   string
	idleSeconds int64
	keysLen     int
	mergeLen    int
	noExpire    bool
	output      string

	buildTime string
	gitHash   string

	//go:embed usage.txt
	usage string
)

func init() {
	flag.StringVar(&redisUrl, "u", "redis://127.0.0.1:6379/0", "")
	flag.StringVar(&separator, "s", "", "")
	flag.Int64Var(&idleSeconds, "i", 86400*7, "")
	flag.IntVar(&keysLen, "sn", 10, "")
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

	if separator == "" || idleSeconds <= 0 || keysLen <= 0 || mergeLen <= 0 || output == "" {
		flag.Usage()
		return
	}

	// set max cpu core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// init idler
	idler, err := NewIdler(redisUrl, separator, idleSeconds, keysLen, mergeLen, output)

	if err != nil {
		log.Fatalf("Fatal Error: init idler failed, redis url '%s', output '%s', %s", redisUrl, output, err)
	}

	// do parse
	err = idler.Run(noExpire)

	if err != nil {
		log.Fatalf("Fatal Error: parse idle data failed, %s", err)
	}

	// save report
	err = idler.Save()

	if err != nil {
		log.Fatalf("Fatal Error: save report failed, %s", err)
	}
}
