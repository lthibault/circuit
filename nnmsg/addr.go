package nnmsg

import (
	"net"
	"net/url"
	"strings"

	"github.com/go-mangos/mangos"
	"github.com/pkg/errors"
)

// Addr represents a nanomsg address
type Addr interface {
	net.Addr
	Protocol() string
}

// StarAddr is an address to a star-socket
type StarAddr struct {
	Trans mangos.Transport
	URL   []*url.URL
}

// Protocol reveals the protocol
func (s StarAddr) Protocol() string {
	return "star"
}

// Network reveals the scheme
func (s StarAddr) Network() string {
	return "nnmsg"
}

// Slice of addresses
func (s StarAddr) Slice() []string {
	strs := make([]string, len(s.URL))
	for i, u := range s.URL {
		strs[i] = u.String()
	}
	return strs
}

// String representation of the address
func (s StarAddr) String() string {
	return strings.Join(s.Slice(), ",")
}

// DialAll ()
func (s StarAddr) DialAll(sock mangos.Socket) (err error) {
	for _, addr := range s.URL {
		var e error
		if e = sock.Dial(addr.String()); e != nil {
			err = errors.Wrap(err, e.Error())
		}
	}
	return
}
