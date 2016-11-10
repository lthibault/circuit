// Package boot contains the necessary primitives for booting a circuit peer
// and joining the cluster
package boot

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lthibault/circuit/tissue"
	"github.com/lthibault/circuit/tissue/locus"
	"github.com/lthibault/circuit/use/circuit"
	"github.com/lthibault/circuit/use/n"
)

// CircuitServer service struct
type CircuitServer struct {
	ChErr    chan<- error
	SyncFunc func(n.Addr, *tissue.Kin)

	ServerAddr n.Addr
	JoinAddr   n.Addr

	// runtime
	ServiceName string
	LocusName   string
	kill        chan os.Signal
	err         chan<- error
}

// Serve circuitry
func (c CircuitServer) Serve() {
	// tissue + locus
	kin, xkin, rip := tissue.NewKin()
	xlocus := locus.NewLocus(kin, rip)

	// joining
	switch {
	case c.JoinAddr != nil:
		kin.ReJoin(c.JoinAddr)
	case c.SyncFunc != nil:
		go func() {
			// continuously announce the server's address & kin
			// this allows the transponder to restart gracefully
			c.SyncFunc(c.ServerAddr, kin)
		}()
	default:
		log.Println("Singleton server.")
	}

	circuit.Listen(c.ServiceName, xkin)
	circuit.Listen(c.LocusName, xlocus)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	c.Stop()
}

// Stop serving and shut-down the app
func (c CircuitServer) Stop() {
	// close mangos sockets here
	// c.err <- c.tranc.Close()
	c.ChErr <- nil
}
