package store

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	// db dir suffix we perform transactions in
	dbLogs         = "logs"
	dbConf         = "conf"
	ErrKeyNotFound = errors.New("not found")
)

// LevelDBStore implements LogStore and StableStore with go leveldb.
type LevelDBStore struct {
	// an open db, default isolation level: snapshot, read/batch write is atomic
	ldb *leveldb.DB
	// path to store data
	path  string
	rwMtx sync.RWMutex
}

func NewLevelDBStableLogStore(opts ...option) (*LevelDBStore, error) {

	conf := defaultOptions()

	for _, opt := range opts {
		opt(&conf)
	}

	path := filepath.Join(conf.path, dbConf)
	return New(path, conf.ldbOptions)
}

func NewLevelDBCommitLogStore(opts ...option) (*LevelDBStore, error) {

	conf := defaultOptions()
	for _, opt := range opts {
		opt(&conf)
	}

	path := filepath.Join(conf.path, dbLogs)
	return New(path, conf.ldbOptions)

}

func New(path string, o *opt.Options) (*LevelDBStore, error) {
	store, err := leveldb.OpenFile(path, o)
	if err != nil {
		return nil, err
	}
	return &LevelDBStore{path: path, ldb: store}, nil
}

func (ls *LevelDBStore) Close() error {
	return ls.ldb.Close()
}

// Set implements StableStore
func (ls *LevelDBStore) Set(key []byte, val []byte) error {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	return ls.ldb.Put(key, val, nil)
}

// Get returns the value for key, or an empty byte slice if key was not found.
// StableStore
func (ls *LevelDBStore) Get(key []byte) ([]byte, error) {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	val, err := ls.ldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}
	return val, err
}

// SetUint64 implements StableStore
func (ls *LevelDBStore) SetUint64(key []byte, val uint64) error {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	return ls.ldb.Put(key, uint64ToBytes(val), nil)
}

// GetUint64 returns the uint64 value for key, or 0 if key was not found.
// StableStore
func (ls *LevelDBStore) GetUint64(key []byte) (uint64, error) {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	val, err := ls.ldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, ErrKeyNotFound
		}
		return 0, err
	}
	return bytesToUint64(val), nil
}

// FirstIndex returns the first index written. 0 for no entries.
// LogStore.
func (ls *LevelDBStore) FirstIndex() (uint64, error) {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()

	iter := ls.ldb.NewIterator(nil, nil)
	defer iter.Release()
	if iter.First() {
		return bytesToUint64(iter.Key()), nil
	}
	return 0, nil
}

// LastIndex returns the last index written. 0 for no entries.
func (ls *LevelDBStore) LastIndex() (uint64, error) {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	iter := ls.ldb.NewIterator(nil, nil)
	defer iter.Release()
	if iter.Last() {
		key := iter.Key()
		return bytesToUint64(key), nil
	}
	return 0, nil
}

// GetLog gets a log entry at a given index.
func (ls *LevelDBStore) GetLog(index uint64, log *raft.Log) error {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	val, err := ls.ldb.Get(uint64ToBytes(index), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return raft.ErrLogNotFound
		}
		return err
	}
	return decodeMsgPack(val, log)
}

// StoreLog stores a log entry.
func (ls *LevelDBStore) StoreLog(log *raft.Log) error {
	return ls.StoreLogs([]*raft.Log{log})
}

// StoreLogs stores multiple log entries.
func (ls *LevelDBStore) StoreLogs(logs []*raft.Log) error {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	b := leveldb.Batch{}
	for _, log := range logs {
		key := uint64ToBytes(log.Index)
		val, err := encodeMsgPack(log)
		if err != nil {
			return err
		}
		b.Put(key, val.Bytes())
	}

	return ls.ldb.Write(&b, nil)
}

// DeleteRange deletes a range of log entries. The range is inclusive.
// LogStore
func (ls *LevelDBStore) DeleteRange(min uint64, max uint64) error {
	ls.rwMtx.Lock()
	defer ls.rwMtx.Unlock()
	r := &util.Range{
		Start: uint64ToBytes(min),
		Limit: uint64ToBytes(max + 1),
	}

	iter := ls.ldb.NewIterator(r, nil)
	defer iter.Release()

	batch := new(leveldb.Batch)
	for iter.Next() {
		batch.Delete(iter.Key())
	}

	return ls.ldb.Write(batch, nil)
}
