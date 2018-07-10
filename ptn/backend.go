// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package eth implements the PalletOne protocol.
package ptn

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/studyzy/go-palletone/common"
	"github.com/studyzy/go-palletone/common/bloombits"
	"github.com/studyzy/go-palletone/common/event"
	"github.com/studyzy/go-palletone/common/log"
	"github.com/studyzy/go-palletone/common/p2p"
	"github.com/studyzy/go-palletone/common/rpc"
	"github.com/studyzy/go-palletone/configure"
	"github.com/studyzy/go-palletone/core"
	"github.com/studyzy/go-palletone/consensus"
	"github.com/studyzy/go-palletone/core/accounts"
	"github.com/studyzy/go-palletone/core/node"
	"github.com/studyzy/go-palletone/core/types"
	"github.com/studyzy/go-palletone/dag/coredata"
	"github.com/studyzy/go-palletone/internal/ethapi"
	"github.com/studyzy/go-palletone/ptn/downloader"
	"github.com/studyzy/go-palletone/ptn/filters"
	"github.com/studyzy/go-palletone/ptn/gasprice"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *coredata.ChainIndexer)
}

// PalletOne implements the PalletOne full node service.
type PalletOne struct {
	config *Config

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the PalletOne

	// Handlers
	txPool          *coredata.TxPool
	protocolManager *ProtocolManager

	eventMux       *event.TypeMux
	engine         core.ConsensusEngine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *coredata.ChainIndexer         // Bloom indexer operating during block imports

	ApiBackend *EthApiBackend
	gasPrice   *big.Int
	etherbase  common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new PalletOne object (including the
// initialisation of the common PalletOne object)
func New(ctx *node.ServiceContext, config *Config) (*PalletOne, error) {
	//if config.SyncMode == downloader.LightSync {
	//	return nil, errors.New("can't run eth.PalletOne in light sync mode, use les.LightEthereum")
	//}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}

	/*_, genesisErr := gen.SetupGenesisBlock(config.Genesis)
	if _, ok := genesisErr.(*configure.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}*/

	eth := &PalletOne{
		config:         config,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx),
		shutdownChan:  make(chan bool),
		networkId:     config.NetworkId,
		gasPrice:      config.GasPrice,
		etherbase:     config.Etherbase,
		bloomRequests: make(chan chan *bloombits.Retrieval),
		bloomIndexer:  NewBloomIndexer(configure.BloomBitsBlocks),
	}

	log.Info("Initialising PalletOne protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eth.txPool = coredata.NewTxPool(config.TxPool)

	var err error
	if eth.protocolManager, err = NewProtocolManager(config.SyncMode, config.NetworkId, /*eth.eventMux,*/ eth.txPool, eth.engine); err != nil {
		log.Error("NewProtocolManager err:", err)
		return nil, err
	}

	eth.ApiBackend = &EthApiBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)
	return eth, nil
}
/*
// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ptndb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ptndb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}
*/
//CreateConsensusEngine creates the required type of consensus engine instance for an PalletOne service
func CreateConsensusEngine(ctx *node.ServiceContext) core.ConsensusEngine {
	engine := consensus.New()
	return engine
}

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *PalletOne) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			//Service:   NewPublicMinerAPI(s),
			Public: true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			//Service:   NewPrivateMinerAPI(s),
			Public: false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			//Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *PalletOne) ResetWithGenesisBlock(gb *types.Block) {
	//s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *PalletOne) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.etherbase = etherbase
			s.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("etherbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *PalletOne) SetEtherbase(etherbase common.Address) {
	self.lock.Lock()
	self.etherbase = etherbase
	self.lock.Unlock()
}

func (s *PalletOne) AccountManager() *accounts.Manager { return s.accountManager }
func (s *PalletOne) TxPool() *coredata.TxPool          { return s.txPool }
func (s *PalletOne) EventMux() *event.TypeMux          { return s.eventMux }

func (s *PalletOne) Engine() core.ConsensusEngine       { return s.engine }
func (s *PalletOne) IsListening() bool                  { return true } // Always listening
func (s *PalletOne) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *PalletOne) NetVersion() uint64                 { return s.networkId }
func (s *PalletOne) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *PalletOne) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// PalletOne protocol implementation.
func (s *PalletOne) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers

	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// PalletOne protocol.
func (s *PalletOne) Stop() error {
	s.bloomIndexer.Close()
	s.protocolManager.Stop()
	s.txPool.Stop()
	//	s.engine.Stop()
	s.eventMux.Stop()
	close(s.shutdownChan)

	return nil
}
