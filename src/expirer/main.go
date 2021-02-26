package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-redis/redis"
	"github.com/marsmay/golib/flag2"
	"github.com/marsmay/golib/math2"
	"github.com/marsmay/golib/strings2"
)

const ScanBatchNum = 500

var (
	redisUrl   string
	keyPrefixs flag2.Strings
	keyExpires flag2.Integers
	limit      int
	pika       bool
	buildTime  string
	gitHash    string
)

func init() {
	flag.StringVar(&redisUrl, "u", "redis://127.0.0.1:6379/0", "redis url, like [redis|rediss]://[:password@]host[:port][/database]")
	flag.Var(&keyPrefixs, "p", "key prefix, required, can specify multiple")
	flag.Var(&keyExpires, "e", "key expire seconds, required, can specify multiple, must match prefix")
	flag.IntVar(&limit, "l", 0, "processed item limit num (default 0)")
	flag.BoolVar(&pika, "pika", false, "instance is pika (default false)")

	flag.Usage = func() {
		fmt.Printf(`redis-expirer version %s, build at %s
Copyright (C) 2015-2021 by Zivn.
Web site: https://may.ltd/

redis-expirer can set the specified prefix key's expiration to specified seconds.

Usage: redis-expirer [-u url] -p prefix [-p prefix]... -e expire [-e expire]... [-l limit] [-pika] 

Supported redis URLs are in any of these formats:
  redis://[:PASSWORD@]HOST[:PORT][/DATABASE]
  rediss://[:PASSWORD@]HOST[:PORT][/DATABASE]

Options
  -u	 redis url (default: redis://127.0.0.1:6379/0)
  -p	 key prefix, can specify multiple 
  -e	 key expire seconds, can specify multiple, must match prefix 
  -l 	 maximum number of items to be processed, 0 means no limit (default: 0)
  -pika  instance is pika (default: false)

`, gitHash, buildTime)
	}
}

func main() {
	// parse flag
	flag.Parse()

	if len(keyPrefixs) == 0 || len(keyPrefixs) != len(keyExpires) || limit < 0 {
		flag.Usage()
		return
	}

	expires := make(map[string]int, len(keyPrefixs))

	for i := 0; i < len(keyPrefixs); i++ {
		expires[keyPrefixs[i]] = keyExpires[i]
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

		var (
			matchKeys                []string
			matchExpires, setExpires map[string]int
		)

		if len(keys) > 0 {
			matchExpires = make(map[string]int, len(keys))

			if len(keyPrefixs) > 1 {
				matchKeys = make([]string, 0, len(keys))

				for _, key := range keys {
					if ok, prefix := strings2.HasPrefixs(key, keyPrefixs); ok {
						matchKeys = append(matchKeys, key)
						matchExpires[key] = expires[prefix]
					}
				}
			} else {
				matchKeys = keys

				for _, key := range keys {
					matchExpires[key] = keyExpires[0]
				}
			}
		}

		if len(matchKeys) > 0 {
			setExpires = make(map[string]int, len(matchKeys))

			for _, key := range matchKeys {
				var idle, ttl time.Duration

				if !pika {
					idle, err = client.ObjectIdleTime(key).Result()

					if err == redis.Nil {
						continue
					}

					if err != nil {
						log.Fatalf("Fatal Error: get key info failed, key '%s', %s", key, err)
					}
				}

				ttl, err = client.TTL(key).Result()

				if err != nil {
					log.Fatalf("Fatal Error: get key info failed, key '%s', %s", key, err)
				}

				if ttl == -1*time.Second {
					setExpires[key] = math2.Max(0, matchExpires[key]-int(idle/time.Second))
				}
			}
		}

		if len(setExpires) > 0 {
			for key, expire := range setExpires {
				err = client.Expire(key, time.Duration(expire)*time.Second).Err()

				if err != nil {
					log.Fatalf("Fatal Error: expire key failed, key '%s', expire '%d', %s", key, expire, err)
				}

				fmt.Printf("%s, %d\n", key, expire)
			}

			processed += len(setExpires)
		}

		if cursor == 0 {
			break
		}

		if limit > 0 && processed > limit {
			break
		}
	}
}
