package assemble

import (
	"encoding/json"
	"log"
	"net"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/star"
	"github.com/go-mangos/mangos/transport/all"
	"github.com/lthibault/circuit/kit/xor"
	"github.com/pkg/errors"
)

// TransponderFromAddrString takes a string containing a URL or a comma-separated
// list of URLs and returns a transponder
func TransponderFromAddrString(addr string) (*Transponder, error) {
	var u *url.URL
	var err error

	rawAddrs := strings.Split(addr, ",")
	urls := make([]*url.URL, len(rawAddrs))

	for i, ra := range rawAddrs {
		u, err = url.Parse(ra)
		if err != nil {
			return nil, errors.Wrapf(err, "url for discovery and assembly does not parse (%s)", err)
		}
		urls[i] = u
	}

	var multicast Multicaster
	if len(urls) == 1 {
		u = urls[0]
		if u.Scheme == "udp" {
			// udp
			addr, e := net.ResolveUDPAddr(u.Scheme, u.Host)
			if e != nil {
				return nil, errors.Wrapf(err, "udp multicast address does not parse (%s)", e)
			}

			if multicast, err = NewUDPMulticaster(addr); err != nil {
				return nil, err
			}

		} else {
			// nnmsg
			if multicast, err = NewNNMulticaster(u); err != nil {
				return nil, err
			}

		}
	} else if len(urls) > 0 {
		// nnmsg
		if multicast, err = NewNNMulticaster(urls...); err != nil {
			return nil, err
		}
	} else {
		log.Fatalf("invalid discovery address %s", addr)
	}
	return NewTransponder(multicast), nil
}

// Multicaster {}
type Multicaster interface {
	Addr() string
	Recv() ([]byte, error)
	Send([]byte) error
	Close() error
}

// Transponder {}
type Transponder struct {
	ch chan []byte
	m  Multicaster
}

// NewTransponder ()
func NewTransponder(m Multicaster) *Transponder {
	return &Transponder{ch: make(chan []byte), m: m}
}

// Addr returns the underlying Multicaster's address
func (t Transponder) Addr() string {
	return t.m.Addr()
}

// NewScatter ()
func (t Transponder) NewScatter(key xor.Key, payload []byte) *Scatter {
	go func(buf <-chan []byte) {
		if err := t.m.Send(<-buf); err != nil {
			log.Printf("multicast scatter error: " + err.Error())
		}
	}(t.ch)

	// if _, err = t.Send(buf); err != nil {
	// 	log.Printf("multicast scatter error: " + err.Error())
	// }

	return &Scatter{
		scatter: t.ch,
		key:     key,
		payload: payload,
	}
}

// NewGather ()
func (t Transponder) NewGather() *Gather {
	chMsg := make(chan *Msg)
	g := &Gather{
		gather:  t.ch,
		recvMsg: chMsg,
	}

	runtime.SetFinalizer(g, func(g2 *Gather) {
		_ = t.m.Close()
		close(t.ch)
	})

	go func(ch <-chan *Msg) {
		for {
			buf, err := t.m.Recv()
			if err != nil {
				panic(err)
			}
			var msg Msg
			if err = json.Unmarshal(buf, &msg); err != nil {
				continue // malformed invitation
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

// NewUDPMulticaster ()
func NewUDPMulticaster(addr *net.UDPAddr) (Multicaster, error) {
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
	addr []*url.URL
	sock mangos.Socket
}

// NewNNMulticaster ()
func NewNNMulticaster(addr ...*url.URL) (Multicaster, error) {
	var sock mangos.Socket
	var err error
	if sock, err = star.NewSocket(); err != nil {
		return nil, errors.Wrapf(err, "error creating socket: %v", err)
	}

	all.AddTransports(sock)

	var nerr int
	for _, u := range addr {
		if err = sock.Dial(u.String()); err != nil {
			log.Println(errors.Wrapf(err, "error dialing socket %s: %v", u, err))
			nerr++
		}
	}

	if nerr >= len(addr) {
		return nil, errors.New("transponder failed to establish connection")
	}

	return &nnmsg{addr: addr, sock: sock}, nil
}

// String representation of the address
func (n nnmsg) Addr() string {
	return strings.Join(n.Slice(), ",")
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

// Slice of addresses
func (n nnmsg) Slice() []string {
	urls := make([]string, len(n.addr))
	for i, u := range n.addr {
		urls[i] = u.String()
	}
	return urls
}
