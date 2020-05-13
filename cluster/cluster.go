package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/00arthur00/leveldbraft/config"
	"github.com/00arthur00/leveldbraft/store"
	"github.com/hashicorp/consul/agent/consul"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

const (
	ENABLE_WRITE_TRUE  = int32(1)
	ENABLE_WRITE_FALSE = int32(0)
)

type Node interface {
	// Set key/value pair to the cluster.
	Set(key, value string) error

	// Delete key from the distributed storage.
	Delete(key string) error

	// Get Key related value.
	Get(key string) (string, bool)

	// Join remote peer to this cluster.
	Join(peer string) error

	// IsLeader returns whether this node is leader.
	IsLeader() bool

	// Members get members of the cluster
	Members() raft.Configuration
}

type RaftNodeInfo struct {
	raft           *raft.Raft
	fsm            *store.FSM
	leaderNotifyCh chan bool
	log            hclog.Logger
	cache          store.Cacher
	enableWrite    int32
}

// Set key/value pair to the cluster.
func (r *RaftNodeInfo) Set(key string, value string) error {
	logEntry := store.LogEntryData{
		Op:    store.OPSet,
		Key:   key,
		Value: value,
	}

	encodeBytes, err := json.Marshal(&logEntry)
	if err != nil {
		r.log.Error("marshal error:", err)
		return err
	}
	applyFuture := r.raft.Apply(encodeBytes, 5*time.Second)
	if err := applyFuture.Error(); err != nil {
		r.log.Error("raft.apply: %#v", err)
		return err
	}
	return nil
}

// Delete key from the distributed storage
func (r *RaftNodeInfo) Delete(key string) error {
	kv := &store.LogEntryData{
		Op:  store.OPDel,
		Key: key,
	}

	encodeBytes, err := json.Marshal(&kv)
	if err != nil {
		r.log.Error("marshal error:%#v", err)
		return err
	}
	applyFuture := r.raft.Apply(encodeBytes, 5*time.Second)
	if err := applyFuture.Error(); err != nil {
		r.log.Error("raft.apply: %#v", err)
		return err
	}
	return nil
}

// Get Key related value
func (r *RaftNodeInfo) Get(key string) (string, bool) {
	value, ok := r.cache.Get(key)
	return value, ok
}

func (r *RaftNodeInfo) Members() raft.Configuration {
	confFutrue := r.raft.GetConfiguration()
	return confFutrue.Configuration()
}

// join cluster with leader and local addr,this runs on server side.
func (r *RaftNodeInfo) Join(peer string) error {
	future := r.raft.AddVoter(raft.ServerID(peer), raft.ServerAddress(peer), 0, 0)
	if err := future.Error(); err != nil {
		r.log.Error("err", err)
		return err
	}
	return nil
}

func (r *RaftNodeInfo) IsLeader() bool {
	return ENABLE_WRITE_TRUE == atomic.LoadInt32(&r.enableWrite)
}
func (r *RaftNodeInfo) MonitorLeadship() {
	//monitor leadship
	for {
		select {
		case leader := <-r.leaderNotifyCh:
			if leader {
				atomic.StoreInt32(&r.enableWrite, ENABLE_WRITE_TRUE)
				r.log.Info("ms", "become leader enable write api")
			} else {
				atomic.StoreInt32(&r.enableWrite, ENABLE_WRITE_FALSE)
				r.log.Info("ms", "become follower disable write api")
			}
		}
	}
}
func newTransport(raftTCPADDR string) (*raft.NetworkTransport, error) {
	addr, err := net.ResolveTCPAddr("tcp", raftTCPADDR)
	if err != nil {
		return nil, err
	}
	transLayer := consul.NewRaftLayer(addr, addr, nil, nil)
	return raft.NewNetworkTransport(transLayer, 3, 10*time.Second, os.Stderr), nil
}

func NewRaftNode(c *config.Config) (*RaftNodeInfo, error) {

	//raft配置
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(c.RaftTCPAddr)
	raftConfig.Logger = hclog.Default()
	raftConfig.SnapshotInterval = 20 * time.Second
	raftConfig.SnapshotThreshold = 2
	leaderNotifyCh := make(chan bool, 1)
	raftConfig.NotifyCh = leaderNotifyCh

	//transport
	transport, err := newTransport(c.RaftTCPAddr)
	if err != nil {
		return nil, err
	}

	//目录创建
	if err := os.MkdirAll(c.DataDir, 0700); err != nil {
		return nil, err
	}

	//fsm
	cache := store.NewCache()
	fsm := store.NewFSM(cache, hclog.Default())

	//snapshotstore & logstore & stablestore
	snapshotStore, err := raft.NewFileSnapshotStore(c.DataDir, 1, os.Stderr)
	if err != nil {
		return nil, err
	}
	logstore, err := store.NewLevelDBCommitLogStore(store.WithPath(c.DataDir))
	if err != nil {
		return nil, fmt.Errorf("new commit log %w", err)
	}
	stablestore, err := store.NewLevelDBStableLogStore(store.WithPath(c.DataDir))
	if err != nil {
		return nil, fmt.Errorf("new stable log %w", err)
	}

	//raftnode
	raftNode, err := raft.NewRaft(raftConfig, fsm, logstore, stablestore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	//boostrap
	if c.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		// if err := raft.BootstrapCluster(raftConfig, logstore, stablestore, snapshotStore, transport, configuration); err != nil {
		future := raftNode.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			return nil, err
		}
	}

	node := &RaftNodeInfo{
		raft:           raftNode,
		fsm:            fsm,
		leaderNotifyCh: leaderNotifyCh,
		cache:          cache,
		log:            hclog.Default(),
	}
	go node.MonitorLeadship()
	return node, nil
}

func JoinCluster(c *config.Config) error {

	url := fmt.Sprintf("http://%s/join?peer=%s", c.JoinAddr, c.RaftTCPAddr)

	resp, err := http.Get(url)
	if resp != nil {
		defer func() {
			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return errors.New(fmt.Sprintf("Error joining cluster %s", body))
	}

	return nil
}
