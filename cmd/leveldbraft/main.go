package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/00arthur00/leveldbraft/cluster"
	"github.com/00arthur00/leveldbraft/config"
	"github.com/emicklei/go-restful"
	"github.com/hashicorp/go-hclog"
	"github.com/oklog/oklog/pkg/group"
)

var conf config.Config

func main() {
	flag.StringVar(&conf.HTTPAddr, "httpaddr", ":8901", "http addr to listen")
	flag.BoolVar(&conf.Bootstrap, "bootstrap", true, "start as raft cluster leader")
	flag.StringVar(&conf.JoinAddr, "join", "", "join addr for raft cluster")
	flag.StringVar(&conf.DataDir, "datadir", "./leveldb", "data directory")
	flag.StringVar(&conf.RaftTCPAddr, "raft", ":8902", "raft tcp addr")
	flag.Parse()

	//new raft node
	node, err := cluster.NewRaftNode(&conf)
	if err != nil {
		hclog.Default().Error("new raft node", err)
		os.Exit(1)
	}

	//join cluster
	if conf.JoinAddr != "" {
		if err := cluster.JoinCluster(&conf); err != nil {
			hclog.Default().Error("join cluster", err)
			return
		}
	}

	g := group.Group{}

	//http server.
	{
		c := restful.NewContainer()
		c.Add(cluster.NewWebService(node, hclog.Default()))
		//swagger api
		registerOpenAPI(c, "")
		//cors
		setcors(c)

		server := &http.Server{Addr: conf.HTTPAddr, Handler: c}
		g.Add(func() error {
			hclog.Default().Info("http listening on ", conf.HTTPAddr)
			return server.ListenAndServe()
		}, func(error) {
			if err := server.Shutdown(context.TODO()); err != nil {
				hclog.Default().Error("shutdown server with error: ", err.Error)
			}
		})
	}

	// terminator
	{
		ctx, cancel := context.WithCancel(context.Background())
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		g.Add(func() error {
			select {
			case s := <-c:
				hclog.Default().Info("Exiting... caught signal ", s)
			case <-ctx.Done():
			}
			return nil
		}, func(error) {
			cancel()
		})
	}

	if err := g.Run(); err != nil {
		hclog.Default().Error("shutdown with error:", err.Error())
		os.Exit(2)
	}
	hclog.Default().Info("server gracefully shutdown")
}
