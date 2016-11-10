// Package boot contains the necessary primitives for booting a circuit peer
// and joining the cluster
package boot

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lthibault/circuit/kit/assemble"
	"github.com/lthibault/circuit/tissue"
	"github.com/lthibault/circuit/tissue/locus"
	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"
	"github.com/thejerf/suture"
)

// svcname string, locname string, join n.Addr, addr n.Addr, trans *assemble.Transponder

// ServerConfig stores server configuration
type ServerConfig struct {
	ServiceName, LocusName string
	ServerAddr, JoinAddr   n.Addr
	T                      *assemble.Transponder
}

// NewService initializes a service from a config file
func (cfg ServerConfig) NewService(cherr chan<- error) suture.Service {
	return &server{
		ChErr: cherr, // sub-implement
		T:     cfg.T,

		ServiceName: cfg.ServiceName,
		LocusName:   cfg.LocusName,
		JoinAddr:    cfg.JoinAddr,
		ServerAddr:  cfg.ServerAddr,
	}
}

// server service struct
type server struct {
	ChErr chan<- error
	T     *assemble.Transponder

	ServerAddr n.Addr
	JoinAddr   n.Addr

	// runtime
	ServiceName string
	LocusName   string
	kill        chan os.Signal
	err         chan<- error
}

// Serve circuitry
func (s server) Serve() {
	// tissue + locus
	kin, xkin, rip := tissue.NewKin()
	xlocus := locus.NewLocus(kin, rip)

	// joining
	switch {
	case s.JoinAddr != nil:
		kin.ReJoin(s.JoinAddr)
	case s.T != nil:
		go s.T.Bootstrap(s.ServerAddr, kin)
	default:
		log.Println("Singleton server.")
	}

	circuit.Listen(s.ServiceName, xkin)
	circuit.Listen(s.LocusName, xlocus)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	s.Stop()
}

// Stop serving and shut-down the app
func (s server) Stop() {
	close(s.ChErr)
}
