// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/star"
	"github.com/lthibault/circuit/kit/xor"
	"github.com/lthibault/circuit/nnmsg"
	"github.com/pkg/errors"
)

// Scatter {}
type Scatter struct {
	addr    net.Addr // multicast address
	key     xor.Key
	payload []byte
}

// Msg {}
type Msg struct {
	Key     xor.Key
	Payload []byte
}

// NewScatter : addr is a udp multicast address.
func NewScatter(addr net.Addr, key xor.Key, payload []byte) *Scatter {
	return &Scatter{
		addr:    addr,
		key:     key,
		payload: payload,
	}
}

// Scatter ()
func (s *Scatter) Scatter() {
	var (
		err  error
		sock mangos.Socket
		buff []byte
	)

	chScatter := make(chan []byte)
	defer close(chScatter)

	switch s.addr.(type) {
	case *net.UDPAddr:
		conn, e := net.DialUDP("udp", nil, s.addr.(*net.UDPAddr))
		if e != nil {
			panic(e)
		}

		go func(ch <-chan []byte) {
			defer conn.Close()
			for b := range ch {
				if _, err = conn.Write(b); err != nil {
					log.Printf("multicast scatter error: " + err.Error())
				}
			}
		}(chScatter)

	case *nnmsg.StarAddr:
		addr := s.addr.(*nnmsg.StarAddr)

		if sock, err = star.NewSocket(); err != nil {
			panic(errors.Wrapf(err, "could not discover on addr %s (%v)", addr, err))
		}

		sock.AddTransport(addr.Trans)

		// TODO:  a couple of opportunities for improvement / optimization here
		// 1. STAR Socket created each for each scatter operation (henceforth OP:SCAT)
		//		- only need *one* addr to connect for a given OP:SCAT to work
		// 2. Reuse socket between OP:SCATs ?
		//		- implies abstracting UDP-multicast & STAR-socket behind a "strobe" interface (nomenclature:  think IR strobes)
		for _, a := range addr.Slice() {
			if err = sock.Dial(a); err != nil {
				panic(errors.Wrapf(err, "could not discover on addr %s (%v)", addr, err))
			}
		}

		go func(ch <-chan []byte) {
			defer sock.Close()
			for b := range ch {
				if e := sock.Send(b); e != nil {
					log.Printf("nnmsg scatter error %s", err.Error())
				}
			}
		}(chScatter)

	default:
		panic(fmt.Errorf("(scatter) address type not recognized %s", s.addr))
	}

	msg := &Msg{Key: s.key, Payload: s.payload}
	if buff, err = json.Marshal(msg); err != nil {
		panic(err)
	}

	dur := time.Second
	for i := 0; i < 10; i++ {
		chScatter <- buff
		time.Sleep(dur)
		dur = (dur * 7) / 5
	}
}
