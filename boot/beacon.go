package boot

import (
	"net/url"

	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/star"
	"github.com/go-mangos/mangos/transport/tcp"
	"github.com/pkg/errors"
	"github.com/thejerf/suture"
)

// BeaconConfig {}
type BeaconConfig struct {
	addr string
	sock mangos.Socket
}

// NewService ()
func (b BeaconConfig) NewService(addr string) (s suture.Service, err error) {
	if _, err = url.Parse(addr); err != nil {
		err = errors.Wrap(err, "could not parse beacon bind address")
	}

	b.addr = addr
	s = b
	return
}

// Serve ()
func (b BeaconConfig) Serve() {
	var err error
	if b.sock, err = star.NewSocket(); err != nil {
		panic(errors.Wrap(err, "socket creation failed"))
	}

	b.sock.AddTransport(tcp.NewTransport())

	if err = b.sock.Listen(b.addr); err != nil {
		panic(errors.Wrap(err, "error binding beacon"))
	}

	for {
		if _, err = b.sock.Recv(); err != nil {
			panic(errors.Wrap(err, "socket recv error"))
		}
	}
}

// Stop ()
func (b BeaconConfig) Stop() {
	defer func() {
		b.sock = nil
	}()

	if err := b.sock.Close(); err != nil {
		panic(errors.Wrap(err, "error closing beacon socket"))
	}
}
