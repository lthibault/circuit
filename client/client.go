// Copyright 2013 The Go Circuit Project
// Use of this source code is governed by the license for
// The Go Circuit Project, found in the LICENSE file.
//
// Authors:
//   2013 Petar Maymounkov <p@gocircuit.org>

// Package client provides access to the circuit programming environment to user
// programs.
package client

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/lthibault/circuit/anchor"
	"github.com/lthibault/circuit/client/docker"
	"github.com/lthibault/circuit/kit/assemble"
	_ "github.com/lthibault/circuit/kit/debug/kill" //
	"github.com/lthibault/circuit/sys/lang"
	_ "github.com/lthibault/circuit/sys/tele" //
	"github.com/lthibault/circuit/tissue"
	"github.com/lthibault/circuit/tissue/locus"
	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"

	"github.com/pkg/errors"
)

var _once sync.Once

func _init(key []byte) {
	rand.Seed(time.Now().UnixNano())
	t := n.NewTransport(n.ChooseWorkerID(), &net.TCPAddr{}, key)
	//fmt.Println(t.Addr().String())
	circuit.Bind(lang.New(t))
}

// Client is a live session with a circuit server.
type Client struct {
	y locus.YLocus
}

// Dial establishes a connection to a circuit server specified by a circuit address.
// Circuit addresses are printed to standard output when a server is started with the
// "circuit start …" command.
//
// If authkey is non-nil it is used as a private key and all communications are
// secured by HMAC authentication and RC4 symmetric encryption;
// otherwise transmissions are in plaintext.
//
// Errors in communication, such as a missing server, or invalid URL format
// are reported through panics.
func Dial(addr string, authkey []byte) *Client {
	_once.Do(func() {
		_init(authkey)
	})
	c := &Client{}
	w, err := n.ParseAddr(addr)
	if err != nil {
		panic("circuit address does not parse")
	}
	c.y = locus.YLocus{X: circuit.Dial(w, "locus")}
	return c
}

// DialDiscover dials into an existing circuit, using the discovery system to
// locate peers.
func DialDiscover(multicast string, authkey []byte) *Client {
	var d *assemble.Discovery
	var err error
	if d, err = assemble.NewDiscovery(multicast); err != nil {
		panic(errors.Wrapf(err, "error building Discovery (%v)", err))
	}

	_once.Do(func() {
		_init(authkey)
	})
	c := &Client{}
	dialback := assemble.NewAssembler(circuit.ServerAddr(), d).AssembleClient()
	c.y = locus.YLocus{X: circuit.Dial(dialback, "locus")}
	return c
}

// Addr returns the circuit address of the server that this client is connected to.
func (c *Client) Addr() string {
	return c.y.X.Addr().String()
}

// Walk traverses the global virtual anchor namespace and returns a handle to the desired anchor.
// The first element of walk should be the ID of a live circuit server.
// An up to date list of available circuit servers in the cluster can be obtained by calling View.
// The remainder of the walk slice is up to the user.
// Errors in communication or missing servers are reported as panics.
func (c *Client) Walk(walk []string) Anchor {
	if len(walk) == 0 {
		return c
	}
	p := c.y.GetPeers()[walk[0]]
	if p == nil {
		return nil
	}
	t := c.newTerminal(p.Term, p.Kin)
	return t.Walk(walk[1:])
}

// Path ()
func (c *Client) Path() string {
	return "/"
}

// View returns a map of all currently-live circuit server anchors.
// Errors in communication are reported as panics.
func (c *Client) View() map[string]Anchor {
	var r = make(map[string]Anchor)
	for k, p := range c.y.GetPeers() {
		r[k] = c.newTerminal(p.Term, p.Kin)
	}
	return r
}

func (c *Client) newTerminal(xterm circuit.X, xkin tissue.KinAvatar) terminal {
	return terminal{
		y: anchor.YTerminal{X: xterm},
		k: xkin,
	}
}

// ServerID returns the server ID of the circuit server that this client is connected to.
func (c *Client) ServerID() string {
	return c.y.Self().Key()
}

// MakeChan is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) MakeChan(n int) (Chan, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// MakeProc is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) MakeProc(cmd Cmd) (Proc, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// MakeDocker is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) MakeDocker(run docker.Run) (docker.Container, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// MakeNameserver …
func (c *Client) MakeNameserver(string) (Nameserver, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// MakeOnJoin is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) MakeOnJoin() (Subscription, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// MakeOnLeave is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) MakeOnLeave() (Subscription, error) {
	return nil, errors.New("cannot create elements outside of servers")
}

// Get is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) Get() interface{} {
	return nil
}

// Scrub is an Anchor interface method, not applicable to the root-level anchor.
func (c *Client) Scrub() {}
