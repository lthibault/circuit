// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

package assemble

import (
	"sync"

	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"
)

// DialBack {}
type DialBack struct {
	once sync.Once
	ch   chan n.Addr
}

// NewDialBack ()
func NewDialBack() (*DialBack, *XDialBack) {
	d := &DialBack{ch: make(chan n.Addr, 1)}
	xd := &XDialBack{d}
	return d, xd
}

// ObtainAddr ()
func (d *DialBack) ObtainAddr() n.Addr {
	return <-d.ch
}

// XDialBack {}
type XDialBack struct {
	d *DialBack
}

// OfferAddr ()
func (xd *XDialBack) OfferAddr(addr n.Addr) {
	xd.d.once.Do(func() {
		xd.d.ch <- addr
		close(xd.d.ch)
	})
}

func init() {
	circuit.RegisterValue(&XDialBack{})
}

// YDialBack {}
type YDialBack struct {
	circuit.PermX
}

// OfferAddr ()
func (y YDialBack) OfferAddr(addr n.Addr) {
	defer func() {
		recover()
	}()
	y.Call("OfferAddr", addr)
}
