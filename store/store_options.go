package store

import (
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type options struct {
	path       string
	ldbOptions *opt.Options
}

func defaultOptions() options {
	return options{
		ldbOptions: &opt.Options{
			Filter: filter.NewBloomFilter(10),
		},
	}
}

type option func(o *options)

func WithPath(path string) option {
	return func(o *options) {
		o.path = path
	}
}

func WithLevelDBConf(leveldbConf *opt.Options) option {
	return func(o *options) {
		o.ldbOptions = leveldbConf
	}
}
