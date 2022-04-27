package main

import (
	_ "embed"

	"flag"
	"fmt"
	"log"
	"runtime"

	"github.com/go-redis/redis"
	"github.com/marsmay/golib/flag2"
	"github.com/marsmay/golib/strings2"
)

const ScanBatchNum = 500

var (
	redisUrl   string
	keyPrefixs flag2.Strings
	limit      int

	buildTime string
	gitHash   string

	//go:embed usage.txt
	usage string
)

func init() {
	flag.StringVar(&redisUrl, "u", "redis://127.0.0.1:6379/0", "")
	flag.Var(&keyPrefixs, "p", "")
	flag.IntVar(&limit, "l", 0, "")

	flag.Usage = func() {
		fmt.Printf(usage, gitHash, buildTime)
	}
}

func main() {
	// parse flag
	flag.Parse()

	if len(keyPrefixs) == 0 || limit < 0 {
		flag.Usage()
		return
	}

	// set max cpu core
	runtime.GOMAXPROCS(runtime.NumCPU())

	// init redis client
	options, err := redis.ParseURL(redisUrl)

	if err != nil {
		log.Fatalf("Fatal Error: invalid redis url '%s', %s", redisUrl, err)
	}

	client := redis.NewClient(options)

	// process data
	var (
		pattern   string
		cursor    uint64
		keys      []string
		processed int
	)

	if len(keyPrefixs) > 1 {
		pattern = "*"
	} else {
		pattern = keyPrefixs[0] + "*"
	}

	for {
		keys, cursor, err = client.Scan(cursor, pattern, ScanBatchNum).Result()

		if err != nil {
			log.Fatalf("Fatal Error: scan keys failed, pattern '%s', prefixs '%+v', %s", pattern, keyPrefixs, err)
		}

		var delKeys []string

		if len(keys) > 0 {
			if len(keyPrefixs) > 1 {
				delKeys = make([]string, 0, len(keys))

				for _, key := range keys {
					if ok, _ := strings2.HasPrefixs(key, keyPrefixs); ok {
						delKeys = append(delKeys, key)
					}
				}
			} else {
				delKeys = keys
			}
		}

		if len(delKeys) > 0 {
			err = client.Del(delKeys...).Err()

			if err != nil {
				log.Fatalf("Fatal Error: delete keys failed, keys '%+v', %s", delKeys, err)
			}

			for _, key := range delKeys {
				fmt.Println(key)
			}

			processed += len(delKeys)
		}

		if cursor == 0 {
			break
		}

		if limit > 0 && processed > limit {
			break
		}
	}
}
