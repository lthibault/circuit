// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import (
	"encoding/json"
	"time"

	"github.com/lthibault/circuit/kit/xor"
)

const scatterFrequency = 45 // default: 10

// Msg {}
type Msg struct {
	Key     xor.Key
	Payload []byte
}

// Scatter {}
type Scatter struct {
	scatter chan<- []byte
	key     xor.Key
	payload []byte
}

// Scatter ()
func (s *Scatter) Scatter() {
	defer close(s.scatter)
	var err error
	var buf []byte

	msg := Msg{Key: s.key, Payload: s.payload}
	if buf, err = json.Marshal(msg); err != nil {
		panic(err)
	}

	dur := time.Second
	for i := 0; i < scatterFrequency; i++ {
		s.scatter <- buf
		time.Sleep(dur)
		dur = (dur * 7) / 5
	}
}
