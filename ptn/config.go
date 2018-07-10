// Copyright 2017 The go-ethereum Authors
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

package ptn

import (
	"math/big"
	"os"
	"os/user"
	//"path/filepath"
	//"runtime"
	"time"

	"github.com/studyzy/go-palletone/common"
	"github.com/studyzy/go-palletone/common/hexutil"
	"github.com/studyzy/go-palletone/common/log"
	"github.com/studyzy/go-palletone/configure"
	//"github.com/studyzy/go-palletone/consensus/consensusconfig"
	"github.com/studyzy/go-palletone/contracts/contractconfig"
	"github.com/studyzy/go-palletone/core"
	"github.com/studyzy/go-palletone/dag/coredata"
	"github.com/studyzy/go-palletone/dag/dagconfig"
	"github.com/studyzy/go-palletone/ptn/downloader"
	"github.com/studyzy/go-palletone/ptn/gasprice"
	"github.com/studyzy/go-palletone/vm/vmconfig"
)

// DefaultConfig contains default settings for use on the PalletOne main net.
var DefaultConfig = Config{
	SyncMode:      downloader.FullSync,
	NetworkId:     1,
	LightPeers:    100,
	DatabaseCache: 768,
	TrieCache:     256,
	TrieTimeout:   5 * time.Minute,
	GasPrice:      big.NewInt(0.01 * configure.PalletOne),

	TxPool: coredata.DefaultTxPoolConfig,
	GPO: gasprice.Config{
		Blocks:     20,
		Percentile: 60,
	},
	//Consensus: consensusconfig.DefaultConfig,
	Dag: dagconfig.DefaultConfig,
	Log: log.DefaultConfig,
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	/*would recover
	if runtime.GOOS == "windows" {
		DefaultConfig.Ethash.DatasetDir = filepath.Join(home, "AppData", "Ethash")
	} else {
		DefaultConfig.Ethash.DatasetDir = filepath.Join(home, ".ethash")
	}*/
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the PalletOne main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode
	NoPruning bool

	// Light client options
	LightServ  int `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightPeers int `toml:",omitempty"` // Maximum number of LES client peers

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	TrieCache          int
	TrieTimeout        time.Duration

	// Mining-related options
	Etherbase    common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    []byte         `toml:",omitempty"`
	GasPrice     *big.Int

	// Transaction pool options
	TxPool coredata.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool
	// DAG options
	Dag dagconfig.Config
	//Log config
	Log log.Config
	// Consensus options
	//Consensus consensusconfig.Config
	// VM config
	Vm vmconfig.Config
	//Contract config
	Contract contractconfig.Config

	// Miscellaneous options
	DocRoot string `toml:"-"`
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
