package main

import (
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/marsmay/golib/csv"
	"github.com/marsmay/golib/math2"
	"github.com/marsmay/redis-tools/common"
)

const BarWidth = 64
const ScanBatchNum = 500

type Result struct {
	prefix    string
	kind      string
	num       int64
	idleNum   int64
	idleTime  int64
	idleRatio float64
	ttl       int64
	sample    string
}

type Idler struct {
	client    *redis.Client
	reporter  *csv.Writer
	separator string
	tree      *common.Tree
	results   []*Result
}

func (i *Idler) calcNode(node *common.Node, name string) {
	if node.Childrens == nil {
		result := &Result{
			prefix:  name,
			kind:    node.Kind,
			num:     node.Num,
			idleNum: node.Data["idle_num"],
		}

		if node.Data["idle_num"] > 0 {
			result.idleTime = node.Data["idle_time"] / node.Data["idle_num"]
		}

		if node.Num > 0 {
			result.idleRatio = math2.Percent(node.Data["idle_num"], node.Num, 2)
			result.ttl = node.Data["ttl"] / node.Num
		}

		if len(node.Keys) > 0 {
			result.sample = node.Keys[0]
		}

		i.results = append(i.results, result)
	} else {
		for _, v := range node.Childrens {
			i.calcNode(v, name+i.separator+v.Name)
		}
	}
}

func (i *Idler) Run(noExpire bool) (err error) {
	total, err := i.client.DBSize().Result()

	if err != nil {
		return
	}

	var (
		cursor    uint64
		keys      []string
		processed int64
	)

	for {
		keys, cursor, err = i.client.Scan(cursor, "*", ScanBatchNum).Result()

		if err != nil {
			return
		}

		if len(keys) > 0 {
			var (
				kind      string
				idle, ttl time.Duration
			)

			for _, key := range keys {
				kind, err = i.client.Type(key).Result()

				if err != nil {
					return
				}

				idle, err = i.client.ObjectIdleTime(key).Result()

				if err == redis.Nil {
					err = nil
					continue
				}

				if err != nil {
					return
				}

				ttl, err = i.client.TTL(key).Result()

				if err != nil {
					return
				}

				if !noExpire || ttl == -time.Second {
					i.tree.AddNode(key, kind, map[string]int64{
						"idle": idle.Milliseconds() / 1e3,
						"ttl":  ttl.Milliseconds() / 1e3,
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

func (i *Idler) Save() (err error) {

	for _, node := range i.tree.Nodes {
		i.calcNode(node, node.Name)
	}

	err = i.reporter.WriteLine([]string{"prefix", "type", "num", "idle num", "avg idle", "idle percent", "avg ttl", "sample"})

	if err != nil {
		return
	}

	for _, v := range i.results {
		if v.idleNum > 0 {
			err = i.reporter.WriteLine([]string{
				v.prefix,
				v.kind,
				strconv.FormatInt(v.num, 10),
				strconv.FormatInt(v.idleNum, 10),
				strconv.FormatInt(v.idleTime, 10),
				fmt.Sprintf("%.2f%%", v.idleRatio),
				strconv.FormatInt(v.ttl, 10),
				v.sample,
			})

			if err != nil {
				return
			}
		}
	}

	i.reporter.Close()
	return
}

func NewIdler(url string, separator string, idle int64, keysLen, mergeLen int, output string) (idler *Idler, err error) {
	options, err := redis.ParseURL(url)

	if err != nil {
		return
	}

	fileName := path.Join(output, fmt.Sprintf("keys-%s-%s.csv", options.Addr, time.Now().Format("20060102150405")))
	reporter, err := csv.NewWriter(fileName)

	if err != nil {
		return
	}

	idler = &Idler{
		client:    redis.NewClient(options),
		separator: separator,
		reporter:  reporter,
		tree: common.NewTree(separator, keysLen, mergeLen, func(node *common.Node, data map[string]int64) {
			if data["idle"] > idle {
				node.Data["idle_num"]++
				node.Data["idle_time"] += data["idle"]
			}

			node.Data["ttl"] += data["ttl"]
		}),
		results: make([]*Result, 0, 256),
	}
	return
}
