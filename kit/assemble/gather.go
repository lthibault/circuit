// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import "github.com/lthibault/circuit/kit/xor"

// Gather {}
type Gather struct {
	gather  <-chan []byte
	recvMsg <-chan *Msg
}

// Gather ()
func (s *Gather) Gather() (xor.Key, []byte) {
	msg := <-s.recvMsg
	return msg.Key, msg.Payload
}

// GatherLens {}
type GatherLens struct {
	gather *Gather
	lens   *Lens
}

// NewGatherLens ()
func NewGatherLens(m Multicaster, focus xor.Key, k int) *GatherLens {
	return &GatherLens{
		gather: NewTransponder(m).NewGather(),
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
