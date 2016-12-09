// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import (
	"log"

	"github.com/lthibault/circuit/kit/xor"
	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"
)

// Assembler struct
type Assembler struct {
	focus     xor.Key
	addr      n.Addr     // our circuit address
	multicast *Discovery // discovery
}

// NewAssembler ()
func NewAssembler(addr n.Addr, multicast *Discovery) *Assembler {
	return &Assembler{
		focus:     xor.ChooseKey(),
		addr:      addr,
		multicast: multicast,
	}
}

func (a *Assembler) scatter(origin string) {
	msg := &TraceMsg{
		Origin: origin,
		Addr:   a.addr.String(),
	}

	// send off a sequence of messages announcing our presnence over time
	a.multicast.NewScatter(a.focus, msg.Encode()).Scatter()
}

func (a *Assembler) gather(joinServer JoinFunc) {
	gather := a.multicast.NewGather()
	for {
		_, payload := gather.Gather()
		trace, err := Decode(payload)
		if err != nil {
			log.Printf("Unrecognized trace message (%v)", err)
			continue
		}
		joinAddr, err := n.ParseAddr(trace.Addr)
		if err != nil {
			log.Printf("Trace origin address not parsing (%v)", err)
			continue
		}
		switch trace.Origin {
		case "server":
			joinServer(joinAddr)
		case "client":
			joinClient(a.addr, joinAddr)
		}
	}
}

// JoinFunc ()
type JoinFunc func(n.Addr)

// AssembleServer ()
func (a Assembler) AssembleServer(joinServer JoinFunc) {
	go a.scatter("server")
	go a.gather(joinServer)
}

func joinClient(serverAddr, clientAddr n.Addr) {
	x, err := circuit.TryDial(clientAddr, "dialback")
	if err != nil {
		return
	}
	y := YDialBack{x}
	y.OfferAddr(serverAddr)
}

// AssembleClient ()
func (a Assembler) AssembleClient() n.Addr { // XXX: Clients should get more than one offering.
	d, xd := NewDialBack()
	circuit.Listen("dialback", xd)
	go a.scatter("client")
	return d.ObtainAddr()
}
