package main

import (
	"flag"
	"io"
	"net"

	"github.com/lshpku/TunneLS/remux"
	"github.com/lshpku/TunneLS/remux/client"
	"github.com/lshpku/TunneLS/remux/server"
	"github.com/lshpku/TunneLS/sla"
)

var (
	flagRole          = flag.String("role", "", "must be either \"server\" or \"client\"")
	flagClientNetwork = flag.String("client.net", "tcp", "client bind network type")
	flagClientAddress = flag.String("client.addr", "", "client bind address")
	flagServerNetwork = flag.String("server.net", "tcp", "server bind network type")
	flagServerAddress = flag.String("server.addr", "", "server bind address")
)

func main() {
	flag.Parse()
	sla.Init()
	remux.Init()

	if len(*flagRole) == 0 {
		sla.Fatal("must specify a role")
	}

	var manager remux.Manager
	var lis net.Listener
	var err error

	if *flagRole == "server" {
		manager = server.NewManager()
		lis, err = net.Listen(*flagServerNetwork, *flagServerAddress)
	} else if *flagRole == "client" {
		manager = client.NewManager(*flagServerNetwork, *flagServerAddress))
		lis, err = net.Listen(*flagClientNetwork, *flagClientAddress)
	} else {
		sla.Fatal("unknown role: %s", *flagRole)
	}

	if err != nil {
		sla.Fatal("fail to listen: %s", err.Error())
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			sla.Error("accept: %s", err.Error())
			continue
		}
		manager.Submit(conn)
	}
}
