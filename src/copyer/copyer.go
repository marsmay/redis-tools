package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/marsmay/golib/math2"
)

const ScanBatchNum = 500

type Copyer struct {
	sourceClient *redis.Client
	targetClient *redis.Client
}

func (c *Copyer) copy(sourceKey, targetKey string) (ok bool, ttl time.Duration, err error) {
	ttl, err = c.sourceClient.TTL(sourceKey).Result()

	if err != nil {
		return
	}

	if ttl < 0 {
		ttl = 0
	}

	kind, err := c.sourceClient.Type(sourceKey).Result()

	if err != nil {
		return
	}

	switch strings.ToLower(kind) {
	case "string":
		ok, err = c.copyString(sourceKey, targetKey, ttl)
	case "list":
		ok, err = c.copyList(sourceKey, targetKey, ttl)
	case "set":
		ok, err = c.copySet(sourceKey, targetKey, ttl)
	case "zset":
		ok, err = c.copyZSet(sourceKey, targetKey, ttl)
	case "hash":
		ok, err = c.copyHash(sourceKey, targetKey, ttl)
	}

	return
}

func (c *Copyer) copyString(sourceKey, targetKey string, ttl time.Duration) (ok bool, err error) {
	value, err := c.sourceClient.Get(sourceKey).Result()

	if err == redis.Nil {
		err = nil
		return
	}

	if err != nil {
		return
	}

	err = c.targetClient.Set(targetKey, value, ttl).Err()

	if err != nil {
		return
	}

	ok = true
	return
}

func (c *Copyer) copyList(sourceKey, targetKey string, ttl time.Duration) (ok bool, err error) {
	length, err := c.sourceClient.LLen(sourceKey).Result()

	if err != nil || length == 0 {
		return
	}

	var (
		newlen int64
		values []string
	)

	for i := int64(0); i < length; i += ScanBatchNum {
		values, err = c.sourceClient.LRange(sourceKey, i, i+ScanBatchNum-1).Result()

		if err != nil {
			return
		}

		if len(values) == 0 {
			continue
		}

		params := make([]interface{}, 0, len(values))

		for _, value := range values {
			params = append(params, value)
		}

		newlen, err = c.targetClient.RPush(targetKey, params...).Result()

		if err != nil {
			return
		}
	}

	ok = newlen > 0

	if ok && ttl > 0 {
		err = c.targetClient.Expire(targetKey, ttl).Err()
	}

	return
}

func (c *Copyer) copySet(sourceKey, targetKey string, ttl time.Duration) (ok bool, err error) {
	length, err := c.sourceClient.SCard(sourceKey).Result()

	if err != nil || length == 0 {
		return
	}

	var (
		cursor    uint64
		newlen, n int64
		values    []string
	)

	for {
		values, cursor, err = c.sourceClient.SScan(sourceKey, cursor, "*", math2.MinInt64(length, ScanBatchNum)).Result()

		if err != nil {
			return
		}

		if len(values) > 0 {
			params := make([]interface{}, 0, len(values))

			for _, value := range values {
				params = append(params, value)
			}

			n, err = c.targetClient.SAdd(targetKey, params...).Result()

			if err != nil {
				return
			}

			newlen += n
		}

		if cursor == 0 {
			break
		}
	}

	ok = newlen > 0

	if ok && ttl > 0 {
		err = c.targetClient.Expire(targetKey, ttl).Err()
	}

	return
}

func (c *Copyer) copyZSet(sourceKey, targetKey string, ttl time.Duration) (ok bool, err error) {
	length, err := c.sourceClient.ZCard(sourceKey).Result()

	if err != nil || length == 0 {
		return
	}

	var (
		newlen, n int64
		values    []redis.Z
	)

	for i := int64(0); i < length; i += ScanBatchNum {
		values, err = c.sourceClient.ZRangeWithScores(sourceKey, i, i+ScanBatchNum-1).Result()

		if err != nil {
			return
		}

		if len(values) == 0 {
			continue
		}

		n, err = c.targetClient.ZAdd(targetKey, values...).Result()

		if err != nil {
			return
		}

		newlen += n
	}

	ok = newlen > 0

	if ok && ttl > 0 {
		err = c.targetClient.Expire(targetKey, ttl).Err()
	}

	return
}

func (c *Copyer) copyHash(sourceKey, targetKey string, ttl time.Duration) (ok bool, err error) {
	length, err := c.sourceClient.HLen(sourceKey).Result()

	if err != nil || length == 0 {
		return
	}

	var (
		cursor uint64
		newlen int
		values []string
	)

	for {
		values, cursor, err = c.sourceClient.HScan(sourceKey, cursor, "*", math2.MinInt64(length, ScanBatchNum)).Result()

		if err != nil {
			return
		}

		if len(values) > 0 {
			params := make(map[string]interface{}, len(values)/2)

			for i := 0; i < len(values)-1; i += 2 {
				params[values[i]] = values[i+1]
			}

			err = c.targetClient.HMSet(targetKey, params).Err()

			if err != nil {
				return
			}

			newlen += len(params)
		}

		if cursor == 0 {
			break
		}
	}

	ok = newlen > 0

	if ok && ttl > 0 {
		err = c.targetClient.Expire(targetKey, ttl).Err()
	}

	return
}

func (c *Copyer) Run(sourcePrefix, targetPrefix string) (err error) {
	var (
		cursor uint64
		keys   []string
	)

	for {
		keys, cursor, err = c.sourceClient.Scan(cursor, sourcePrefix+"*", ScanBatchNum).Result()

		if err != nil {
			return
		}

		if len(keys) > 0 {
			for _, key := range keys {
				targetKey := targetPrefix + strings.TrimPrefix(key, sourcePrefix)
				ok, ttl, e := c.copy(key, targetKey)

				if e != nil {
					err = e
					return
				}

				if ok {
					fmt.Printf("%s => %s (%+v)", key, targetKey, ttl)
				}
			}
		}

		if cursor == 0 {
			break
		}
	}

	return
}

func NewCopyer(sourceUrl, targetUrl string) (copyer *Copyer, err error) {
	sourceOpts, err := redis.ParseURL(sourceUrl)

	if err != nil {
		return
	}

	targetOpts, err := redis.ParseURL(targetUrl)

	if err != nil {
		return
	}

	copyer = &Copyer{
		sourceClient: redis.NewClient(sourceOpts),
		targetClient: redis.NewClient(targetOpts),
	}
	return
}
