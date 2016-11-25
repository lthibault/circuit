package boot

import (
	"log"
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
	log.Printf("[ BEACON ] starting peer discovery service on address %s", b.addr)

	var err error
	if b.sock, err = star.NewSocket(); err != nil {
		log.Printf("[ BEACON ] socket creation failed (%s)", err)
	}

	b.sock.AddTransport(tcp.NewTransport())

	if err = b.sock.Listen(b.addr); err != nil {
		log.Printf("[ BEACON ] socket bind error (%s)", err)
	}

	var msg []byte
	for {
		if msg, err = b.sock.Recv(); err != nil {
			log.Printf("[ BEACON ] socket recv error (%s)", err)
		} else {
			log.Printf("[ BEACON ] [ DEBUG ] %s", msg)
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
