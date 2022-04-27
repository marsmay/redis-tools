package main

import (
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/marsmay/golib/csv"
	"github.com/marsmay/golib/math2"
	"github.com/marsmay/redis-tools/common"
)

const BarWidth = 64
const ScanBatchNum = 500
const LenSampleNum = 10

type Result struct {
	prefix string
	kind   string
	num    int64
	keys   []string
	ttl    int64
	sample string
}

type Paser struct {
	client    *redis.Client
	reporter  *csv.Writer
	separator string
	tree      *common.Tree
	results   []*Result
}

func (p *Paser) calcNode(node *common.Node, name string) {
	if node.Childrens == nil {
		result := &Result{
			prefix: name,
			kind:   node.Kind,
			num:    node.Num,
			keys:   node.Keys,
		}

		if node.Num > 0 {
			result.ttl = node.Data["ttl"] / node.Num
		}

		if len(node.Keys) > 0 {
			result.sample = node.Keys[0]
		}

		p.results = append(p.results, result)
	} else {
		for _, v := range node.Childrens {
			p.calcNode(v, name+p.separator+v.Name)
		}
	}
}

func (p *Paser) getStrInfo(keys []string) (itemNum, itemSize int64) {
	itemNums := make([]int64, 0, len(keys))

	for _, key := range keys {
		length, err := p.client.StrLen(key).Result()

		if err != nil {
			log.Printf("Warning: get key length failed, '%s' %s", key, err)
			continue
		}

		if length == 0 {
			continue
		}

		itemNums = append(itemNums, length)
	}

	itemNum = 1
	itemSize = math2.AvgList(itemNums)
	return
}

func (p *Paser) getListInfo(keys []string) (itemNum, itemSize int64) {
	itemNums := make([]int64, 0, len(keys))
	itemSizes := make([]int64, 0, len(keys))

	for _, key := range keys {
		length, err := p.client.LLen(key).Result()

		if err != nil {
			log.Printf("Warning: get list length failed, '%s' %s", key, err)
			continue
		}

		if length == 0 {
			continue
		}

		itemNums = append(itemNums, length)

		items, err := p.client.LRange(key, 0, LenSampleNum).Result()

		if err != nil {
			log.Printf("Warning: get list items failed, '%s' %s", key, err)
			continue
		}

		for _, item := range items {
			itemSizes = append(itemSizes, int64(len(item)))
		}
	}

	itemNum = math2.AvgList(itemNums)
	itemSize = math2.AvgList(itemSizes)
	return
}

func (p *Paser) getSetInfo(keys []string) (itemNum, itemSize int64) {
	itemNums := make([]int64, 0, len(keys))
	itemSizes := make([]int64, 0, len(keys))

	for _, key := range keys {
		length, err := p.client.SCard(key).Result()

		if err != nil {
			log.Printf("Warning: get set length failed, '%s' %s", key, err)
			continue
		}

		if length == 0 {
			continue
		}

		itemNums = append(itemNums, length)

		items, err := p.client.SRandMemberN(key, LenSampleNum).Result()

		if err != nil {
			log.Printf("Warning: get set items failed, '%s' %s", key, err)
			continue
		}

		for _, item := range items {
			itemSizes = append(itemSizes, int64(len(item)))
		}
	}

	itemNum = math2.AvgList(itemNums)
	itemSize = math2.AvgList(itemSizes)
	return
}

func (p *Paser) getZSetInfo(keys []string) (itemNum, itemSize int64) {
	itemNums := make([]int64, 0, len(keys))
	itemSizes := make([]int64, 0, len(keys))

	for _, key := range keys {
		length, err := p.client.ZCard(key).Result()

		if err != nil {
			log.Printf("Warning: get zset length failed, '%s' %s", key, err)
			continue
		}

		if length == 0 {
			continue
		}

		itemNums = append(itemNums, length)

		items, err := p.client.ZRange(key, 0, LenSampleNum).Result()

		if err != nil {
			log.Printf("Warning: get zset items failed, '%s' %s", key, err)
			continue
		}

		for _, item := range items {
			itemSizes = append(itemSizes, int64(len(item)))
		}
	}

	itemNum = math2.AvgList(itemNums)
	itemSize = math2.AvgList(itemSizes)
	return
}

func (p *Paser) getHashInfo(keys []string) (itemNum, itemSize int64) {
	itemNums := make([]int64, 0, len(keys))
	itemSizes := make([]int64, 0, len(keys))

	for _, key := range keys {
		length, err := p.client.HLen(key).Result()

		if err != nil {
			log.Printf("Warning: get zset length failed, '%s' %s", key, err)
			continue
		}

		if length == 0 {
			continue
		}

		itemNums = append(itemNums, length)

		var (
			cursor uint64
			values []string
			num    int
		)

		for {
			values, cursor, err = p.client.HScan(key, cursor, "*", LenSampleNum).Result()

			if err != nil {
				log.Printf("Warning: get hash items failed, '%s' %s", key, err)
				break
			}

			if len(values) > 0 {
				for i := 0; i < len(values)-1; i += 2 {
					itemSizes = append(itemSizes, int64(len(values[i+1])))
					num++
				}
			}

			if num >= LenSampleNum || cursor == 0 {
				break
			}
		}
	}

	itemNum = math2.AvgList(itemNums)
	itemSize = math2.AvgList(itemSizes)
	return
}

func (p *Paser) getLength(kind string, keys []string) (itemNum, itemSize int64) {
	switch strings.ToLower(kind) {
	case "string":
		itemNum, itemSize = p.getStrInfo(keys)
	case "list":
		itemNum, itemSize = p.getListInfo(keys)
	case "set":
		itemNum, itemSize = p.getSetInfo(keys)
	case "zset":
		itemNum, itemSize = p.getZSetInfo(keys)
	case "hash":
		itemNum, itemSize = p.getHashInfo(keys)
	}

	return
}

func (p *Paser) Run(noExpire bool) (err error) {
	total, err := p.client.DBSize().Result()

	if err != nil {
		return
	}

	var (
		cursor    uint64
		keys      []string
		processed int64
	)

	for {
		keys, cursor, err = p.client.Scan(cursor, "*", ScanBatchNum).Result()

		if err != nil {
			return
		}

		if len(keys) > 0 {
			var (
				kind string
				ttl  time.Duration
			)

			for _, key := range keys {
				kind, err = p.client.Type(key).Result()

				if err != nil {
					return
				}

				ttl, err = p.client.TTL(key).Result()

				if err != nil {
					return
				}

				if !noExpire || ttl == -time.Second {
					p.tree.AddNode(key, kind, map[string]int64{
						"ttl": ttl.Milliseconds() / 1e3,
					})
				}

				processed++
			}

			common.ProgressBar(BarWidth, processed, total, "scan keys ...")
		}

		if cursor == 0 {
			break
		}
	}

	fmt.Println()
	return
}

func (p *Paser) Save() (err error) {
	for _, node := range p.tree.Nodes {
		p.calcNode(node, node.Name)
	}

	err = p.reporter.WriteLine([]string{"prefix", "type", "num", "avg item num", "avg item size", "total item num", "total item size", "avg ttl", "sample"})

	if err != nil {
		return
	}

	for _, v := range p.results {
		itemNum, itemSize := p.getLength(v.kind, v.keys)

		err = p.reporter.WriteLine([]string{
			v.prefix,
			v.kind,
			strconv.FormatInt(v.num, 10),
			strconv.FormatInt(itemNum, 10),
			strconv.FormatInt(itemSize, 10),
			strconv.FormatInt(v.num*itemNum, 10),
			strconv.FormatInt(v.num*itemNum*itemSize, 10),
			strconv.FormatInt(v.ttl, 10),
			v.sample,
		})

		if err != nil {
			return
		}
	}

	p.reporter.Close()
	return
}

func NewPaser(url string, separator string, keysLen, mergeLen int, output string) (paser *Paser, err error) {
	options, err := redis.ParseURL(url)

	if err != nil {
		return
	}

	fileName := path.Join(output, fmt.Sprintf("keys-%s-%s.csv", options.Addr, time.Now().Format("20060102150405")))
	reporter, err := csv.NewWriter(fileName)

	if err != nil {
		return
	}

	paser = &Paser{
		client:    redis.NewClient(options),
		separator: separator,
		reporter:  reporter,
		tree: common.NewTree(separator, keysLen, mergeLen, func(node *common.Node, data map[string]int64) {
			node.Data["ttl"] += data["ttl"]
		}),
		results: make([]*Result, 0, 256),
	}
	return
}
