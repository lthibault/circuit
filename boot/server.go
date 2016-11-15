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

// ServerConfig stores server configuration
type ServerConfig struct {
	Discover               string
	ServiceName, LocusName string
	ServerAddr, JoinAddr   n.Addr

	B suture.Service
	T *assemble.Transponder
}

// Circuit initializes a circuit server from a config file
func Circuit(cfg ServerConfig) {
	// tissue + locus
	kin, xkin, rip := tissue.NewKin()
	xlocus := locus.NewLocus(kin, rip)

	// joining
	switch {
	case cfg.JoinAddr != nil:
		kin.ReJoin(cfg.JoinAddr)
	case cfg.T != nil:
		go cfg.T.Bootstrap(cfg.ServerAddr, kin)
	default:
		log.Println("Singleton server.")
	}

	circuit.Listen(cfg.ServiceName, xkin)
	circuit.Listen(cfg.LocusName, xlocus)

	services := suture.NewSimple("circuit-services")
	if cfg.B != nil {
		services.Add(cfg.B)
	}
	if cfg.T != nil {
		services.Add(cfg.T)
	}
	services.ServeBackground()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	services.Stop()
}
