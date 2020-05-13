package store

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

// FSM real data cache in memory
type FSM struct {
	c   Cacher
	log hclog.Logger
}

func NewFSM(c Cacher, log hclog.Logger) *FSM {
	return &FSM{c, log}
}

type OP string

const (
	OPSet OP = "set"
	OPDel OP = "del"
)

func (op OP) String() string {
	return string(op)
}

type LogEntryData struct {
	Op    OP
	Key   string
	Value string
}

// Apply log is invoked once a log entry is committed.
// It returns a value which will be made available in the
// ApplyFuture returned by Raft.Apply method if that
// method was called on the same Raft node as the FSM.
func (fsm *FSM) Apply(logEntry *raft.Log) interface{} {
	var kv LogEntryData
	if err := json.Unmarshal(logEntry.Data, &kv); err != nil {
		panic(fmt.Errorf("failed to apply request: %#v", logEntry))
	}
	var ret interface{}
	switch kv.Op {
	case OPDel:
		fsm.c.Del(kv.Key)
	case OPSet:
		fsm.c.Set(kv.Key, kv.Value)
	}
	fsm.log.Debug("fms.Apply(), logEntry:%s, ret:%v\n", logEntry.Data, ret)
	return ret
}

// Snapshot is used to support log compaction. This call should
// return an FSMSnapshot which can be used to save a point-in-time
// snapshot of the FSM. Apply and Snapshot are not called in multiple
// threads, but Apply will be called concurrently with Persist. This means
// the FSM should be implemented in a fashion that allows for concurrent
// updates while a snapshot is happening.
func (fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &snapshot{c: fsm.c}, nil
}

// Restore is used to restore an FSM from a snapshot. It is not called
// concurrently with any other command. The FSM must discard all previous
// state.
func (fsm *FSM) Restore(old io.ReadCloser) error {
	defer old.Close()
	return fsm.c.UnMarshal(old)
}
