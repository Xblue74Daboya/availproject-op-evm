package main

import (
	"errors"
	"flag"
	"log"
	"time"

	"github.com/0xPolygon/polygon-edge/helper/common"

	golog "github.com/ipfs/go-log/v2"
	"github.com/maticnetwork/avail-settlement/consensus/avail"
	"github.com/maticnetwork/avail-settlement/pkg/config"
	"github.com/maticnetwork/avail-settlement/server"
)

const (
	AvailConsensus server.ConsensusType = "avail"
)

func main() {
	var bootnode bool
	var availAddr, path, accountPath string
	flag.StringVar(&availAddr, "avail-addr", "ws://127.0.0.1:9944/v1/json-rpc", "Avail JSON-RPC URL")
	flag.StringVar(&path, "config-file", "./configs/bootnode.yaml", "Path to the configuration file")
	flag.StringVar(&accountPath, "account-config-file", "./configs/account", "Path to the account mnemonic file")
	flag.BoolVar(&bootnode, "bootstrap", false, "bootstrap flag must be specified for the first node booting a new network from the genesis")

	flag.Parse()

	// Enable LibP2P logging
	golog.SetAllLoggers(golog.LevelDebug)

	config, err := config.NewServerConfig(path)
	if err != nil {
		log.Fatalf("failure to get node configuration: %s", err)
	}

	// Enable TxPool P2P gossiping
	config.Seal = true

	log.Printf("Server config: %+v", config)

	// Attach the concensus to the server
	err = server.RegisterConsensus(AvailConsensus, avail.Factory(avail.Config{Bootnode: bootnode, AvailAddr: availAddr, AccountFilePath: accountPath}))
	if err != nil {
		log.Fatalf("failure to register consensus: %s", err)
	}

	serverInstance, err := server.NewServer(config)
	if err != nil {
		log.Fatalf("failure to start node: %s", err)
	}

	log.Printf("Server instance %#v", serverInstance)

	if err := HandleSignals(serverInstance.Close); err != nil {
		log.Fatalf("handle signal error: %s", err)
	}
}

// HandleSignals is a helper method for handling signals sent to the console
// Like stop, error, etc.
func HandleSignals(closeFn func()) error {
	signalCh := common.GetTerminationSignalCh()
	sig := <-signalCh

	log.Printf("\n[SIGNAL] Caught signal: %v\n", sig)

	// Call the Minimal server close callback
	gracefulCh := make(chan struct{})

	go func() {
		if closeFn != nil {
			closeFn()
		}

		close(gracefulCh)
	}()

	select {
	case <-signalCh:
		return errors.New("shutdown by signal channel")
	case <-time.After(5 * time.Second):
		return errors.New("shutdown by timeout")
	case <-gracefulCh:
		return nil
	}
}