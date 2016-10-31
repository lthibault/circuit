// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/lthibault/circuit/kit/xor"
	"github.com/lthibault/circuit/nnmsg"
)

// Gather {}
type Gather struct {
	addr net.Addr // udp multicast address for discovery
	recv <-chan *Msg
}

// NewGather ()
func NewGather(addr net.Addr) (g *Gather) {
	var err error
	chGather := make(chan *Msg)
	g = &Gather{
		addr: addr,
		recv: chGather,
	}

	switch addr.(type) {
	case *net.UDPAddr:
		var conn *net.UDPConn
		if conn, err = net.ListenMulticastUDP("udp", nil, addr.(*net.UDPAddr)); err != nil {
			log.Printf("problem listening to udp multicast: %v", err)
			os.Exit(1)
		}

		runtime.SetFinalizer(g,
			func(g2 *Gather) {
				conn.Close()
				close(chGather)
			},
		)

		go func(ch chan<- *Msg) {
			buf := make([]byte, 7e3)
			for {
				n, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					panic(err)
				}
				var msg Msg
				if err = json.Unmarshal(buf[:n], &msg); err != nil {
					continue // malformed invitation
				}
				ch <- &msg
			}
		}(chGather)

	case *nnmsg.StarAddr:
		log.Fatal("NOT IMPLEMENTED")
	}
	return
}

// Gather ()
func (s *Gather) Gather() (xor.Key, []byte) {
	msg := <-s.recv
	return msg.Key, msg.Payload
}

// GatherLens {}
type GatherLens struct {
	gather *Gather
	lens   *Lens
}

// NewGatherLens ()
func NewGatherLens(addr net.Addr, focus xor.Key, k int) *GatherLens {
	return &GatherLens{
		gather: NewGather(addr),
		lens:   NewLens(focus, k),
	}
}

func (s *GatherLens) String() string {
	return s.lens.String()
}

// Gather ()
func (s *GatherLens) Gather() []byte {
	for {
		key, payload := s.gather.Gather()
		if s.lens.Remember(key) {
			return payload
		}
	}
}

// Clear ()
func (s *GatherLens) Clear() {
	s.lens.Clear()
}
