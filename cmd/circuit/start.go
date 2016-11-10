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
	"path"

	"github.com/lthibault/circuit/boot"
	"github.com/lthibault/circuit/element/docker"
	"github.com/lthibault/circuit/kit/assemble"
	"github.com/lthibault/circuit/tissue"
	"github.com/lthibault/circuit/use/n"
	"github.com/pkg/errors"
	"github.com/thejerf/suture"

	"github.com/urfave/cli"
)

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

	app := suture.NewSimple("application")

	// peer discovery
	var trans *assemble.Transponder
	if trans, err = assemble.NewTransponder(c.String("discover")); err != nil {
		return errors.Wrapf(err, "error booting transponder (%s)", err)
	}
	app.Add(trans)

	// TODO
	// runtime
	// circ, cherr := boot.NewServer()
	// app.Add(circ)
	// app.ServeBackground()
	// return <-cherr

	chErr := make(chan error)
	cfg := boot.CircuitServer{
		ChErr:    chErr,
		SyncFunc: trans.Bootstrap,

		ServiceName: tissue.ServiceName,
		LocusName:   LocusName,
		JoinAddr:    join,
		ServerAddr:  load(tcpaddr, varDir, readkey(c)),
	}

	app.Add(cfg)

	app.ServeBackground()
	return <-chErr
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

// LocusName ???
const LocusName = "locus"

func dontPanic(call func(), ifPanic string) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("%s (%s)", ifPanic, r)
		}
	}()
	call()
}
