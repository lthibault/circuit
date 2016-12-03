// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

// This package provides the executable program for the resource-sharing circuit app
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/lthibault/circuit/element/docker"
	"github.com/lthibault/circuit/kit/assemble"
	"github.com/lthibault/circuit/tissue"
	"github.com/lthibault/circuit/tissue/locus"
	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"
	"github.com/pkg/errors"

	"github.com/urfave/cli"
)

// LocusName ???
const LocusName = "locus"

func server(c *cli.Context) (err error) {
	println("CIRCUIT 2015 gocircuit.org")

	if c.Bool("docker") {
		cmd, e := docker.Init()
		if e != nil {
			return errors.Wrapf(e, "cannot use docker: %v", e)
		}
		log.Printf("Enabling docker elements, using %s", cmd)
	}

	// parse arguments
	var tcpaddr *net.TCPAddr
	if tcpaddr, err = parseAddr(c); err != nil { // server bind address
		return errors.Wrapf(err, "cannot parse server bind address %s (%s)", tcpaddr, err)
	}

	var join n.Addr // join address of another circuit server
	if c.IsSet("join") {
		if join, err = n.ParseAddr(c.String("join")); err != nil {
			return errors.Wrapf(err, "join address does not parse (%s)", err)
		}
	}

	// server instance working directory
	var varDir string
	if !c.IsSet("var") {
		varDir = path.Join(os.TempDir(), fmt.Sprintf("%s-%%W-P%04d", n.Scheme, os.Getpid()))
	} else {
		varDir = c.String("var")
	}

	// peer discovery
	var trans *assemble.Transponder
	if trans, err = assemble.NewTransponder(c.String("discover")); err != nil {
		return errors.Wrapf(err, "error booting transponder (%s)", err)
	}

	// start circuit runtime
	addr := load(tcpaddr, varDir, readkey(c))

	// tissue + locus
	kin, xkin, rip := tissue.NewKin()
	xlocus := locus.NewLocus(kin, rip)

	// joining
	switch {
	case join != nil:
		kin.ReJoin(join)
	case trans != nil:
		go assemble.NewAssembler(addr, trans).AssembleServer(func(joinAddr n.Addr) {
			kin.ReJoin(joinAddr)
		})
	default:
		log.Println("Singleton server.")
	}

	circuit.Listen(tissue.ServiceName, xkin)
	circuit.Listen(LocusName, xlocus)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	return nil
}

func parseAddr(c *cli.Context) (*net.TCPAddr, error) {
	switch {
	case c.String("addr") != "":
		addr, err := net.ResolveTCPAddr("tcp", c.String("addr"))
		if err != nil {
			return nil, errors.Wrapf(err, "resolve %s (%s)\n", addr, err)
		}
		if len(addr.IP) == 0 {
			addr.IP = net.IPv4zero
		}
		return addr, nil

	case c.String("if") != "":
		ifc, err := net.InterfaceByName(c.String("if"))
		if err != nil {
			return nil, errors.Wrapf(err, "interface %s not found (%v)", c.String("if"), err)
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			return nil, errors.Wrapf(err, "interface address cannot be retrieved (%v)", err)
		}
		if len(addrs) == 0 {
			return nil, errors.New("no addresses associated with this interface")
		}
		for _, a := range addrs { // pick the IPv4 one
			ipn := a.(*net.IPNet)
			if ipn.IP.To4() == nil {
				continue
			}
			return &net.TCPAddr{IP: ipn.IP}, nil
		}
		return nil, errors.New("specified interface has no IPv4 addresses")
	default:
		return nil, errors.New("either an -addr or an -if option is required to start a server")
	}
}
