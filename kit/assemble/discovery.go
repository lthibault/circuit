package assemble

import (
	"encoding/json"
	"log"
	"net"
	"net/url"
	"runtime"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/star"
	"github.com/go-mangos/mangos/transport/all"
	"github.com/lthibault/circuit/kit/xor"
	"github.com/pkg/errors"
)

type multicaster interface {
	Addr() string
	Recv() ([]byte, error)
	Send([]byte) error
	Close() error
}

// Discovery {}
type Discovery struct {
	ch chan []byte
	m  multicaster
}

// NewDiscovery ()
func NewDiscovery(addr string) (*Discovery, error) {
	var err error
	var u *url.URL

	if u, err = url.Parse(addr); err != nil {
		err = errors.Wrapf(
			err, "url for discovery and assembly does not parse (%s)", err,
		)
		return nil, err
	}

	var m multicaster
	switch u.Scheme {
	case "udp":
		addr, e := net.ResolveUDPAddr(u.Scheme, u.Host)
		if e != nil {
			err = errors.Wrapf(err, "udp multicast address does not parse (%s)", e)
			return nil, err
		}

		if m, err = newUDPMulticaster(addr); err != nil {
			return nil, err
		}
	default:
		if m, err = newNNMulticaster(u); err != nil {
			return nil, err
		}
	}
	return &Discovery{
		ch: make(chan []byte),
		m:  m,
	}, nil
}

// Addr returns the underlying Multicaster's address
func (d Discovery) Addr() string {
	return d.m.Addr()
}

// NewScatter ()
func (d Discovery) NewScatter(key xor.Key, payload []byte) *Scatter {
	go func(buf <-chan []byte) {
		if err := d.m.Send(<-buf); err != nil {
			log.Printf("multicast scatter error: " + err.Error())
		}
	}(d.ch)

	return &Scatter{
		scatter: d.ch,
		key:     key,
		payload: payload,
	}
}

// NewGather ()
func (d Discovery) NewGather() *Gather {
	chMsg := make(chan *Msg)
	g := &Gather{
		gather:  d.ch,
		recvMsg: chMsg,
	}

	runtime.SetFinalizer(g, func(g2 *Gather) {
		_ = d.m.Close()
		close(d.ch)
	})

	go func(ch <-chan *Msg) {
		for {
			buf, err := d.m.Recv()
			if err != nil {
				panic(err)
			}
			var msg Msg
			if err = json.Unmarshal(buf, &msg); err != nil {
				log.Println("Malformed invitation:", err) // DEBUG
				continue                                  // malformed invitation
			}
			chMsg <- &msg
		}
	}(chMsg)

	return g
}

type udp struct {
	addr  *net.UDPAddr
	wconn *net.UDPConn
	rconn *net.UDPConn
}

func newUDPMulticaster(addr *net.UDPAddr) (multicaster, error) {
	var (
		wconn *net.UDPConn
		rconn *net.UDPConn
		err   error
	)

	if rconn, err = net.ListenMulticastUDP("udp", nil, addr); err != nil {
		err = errors.Wrapf(err, "problem listening to udp multicast: %v", err)
		return nil, err
	}

	if wconn, err = net.DialUDP("udp", nil, addr); err != nil {
		err = errors.Wrapf(err, "problem listening to udp multicast: %v", err)
		return nil, err
	}

	return &udp{addr: addr, rconn: rconn, wconn: wconn}, nil
}

func (u udp) Addr() string {
	return u.addr.String()
}

func (u udp) Send(b []byte) (err error) {
	var n int
	if n, err = u.wconn.Write(b); n != len(b) {
		err = errors.Wrapf(err, "wrote %d of %d bytes", n, len(b))
	}
	return
}

func (u udp) Recv() ([]byte, error) {
	buf := make([]byte, 7e3)
	n, _, err := u.rconn.ReadFromUDP(buf)
	return buf[:n], err
}

func (u udp) Close() error {
	return u.Close()
}

type nnmsg struct {
	addr *url.URL
	sock mangos.Socket
}

// NewNNMulticaster ()
func newNNMulticaster(addr *url.URL) (multicaster, error) {
	var sock mangos.Socket
	var err error
	if sock, err = star.NewSocket(); err != nil {
		return nil, errors.Wrapf(err, "error creating socket: %v", err)
	}

	all.AddTransports(sock)

	if err = sock.Dial(addr.String()); err != nil {
		return nil, errors.New("Discovery failed to establish connection")
	}

	return &nnmsg{addr: addr, sock: sock}, nil
}

// String representation of the address
func (n nnmsg) Addr() string {
	return n.addr.String()
}

func (n nnmsg) Send(b []byte) error {
	return n.sock.Send(b)
}

func (n nnmsg) Recv() ([]byte, error) {
	return n.sock.Recv()
}

func (n nnmsg) Close() error {
	return n.sock.Close()
}
