// Copyright 2018 PalletOne
// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// gptn is the official command-line client for PalletOne.
package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/studyzy/go-palletone/cmd/console"
	"github.com/studyzy/go-palletone/cmd/utils"
	"github.com/studyzy/go-palletone/common/log"
	"github.com/studyzy/go-palletone/core/accounts"
	"github.com/studyzy/go-palletone/core/accounts/keystore"
	"github.com/studyzy/go-palletone/core/node"
	"github.com/studyzy/go-palletone/internal/debug"
	"github.com/studyzy/go-palletone/ptn"
	"github.com/studyzy/go-palletone/ptnclient"
	"github.com/studyzy/go-palletone/statistics/metrics"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "gptn" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	// The app that holds all commands and flags.
	// 新建一个全局的app结构，用来管理程序启动，命令行配置等
	app = utils.NewApp(gitCommit, "the go-palletone command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.BootnodesV4Flag,
		utils.BootnodesV5Flag,
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.NoUSBFlag,
		utils.DashboardEnabledFlag,
		utils.DashboardAddrFlag,
		utils.DashboardPortFlag,
		utils.DashboardRefreshFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.FastSyncFlag,
		utils.LightModeFlag,
		utils.SyncModeFlag,
		utils.GCModeFlag,
		utils.LightServFlag,
		utils.LightPeersFlag,
		utils.LightKDFFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheGCFlag,
		utils.TrieCacheGenFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.EtherbaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.TargetGasLimitFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DeveloperFlag,
		utils.DeveloperPeriodFlag,
		utils.TestnetFlag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.RPCVirtualHostsFlag,
		utils.EthStatsURLFlag,
		utils.MetricsEnabledFlag,
		utils.FakePoWFlag,
		utils.NoCompactionFlag,
		// utils.GpoBlocksFlag,
		// utils.GpoPercentileFlag,
		utils.ExtraDataFlag,
		utils.DagValue1Flag,
		utils.DagValue2Flag,
		utils.LogValue1Flag,
		utils.LogValue2Flag,
		utils.LogValue3Flag,
		utils.LogValue4Flag,
		utils.LogValue5Flag,
		configFileFlag,
		GenesisJsonPathFlag,
	}

	rpcFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RPCApiFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
	}
)

func init() {
	// 先调用初始化函数，设置app的各个参数
	// Initialize the CLI app and start Gptn
	// gptn处理函数会在 app.HandleAction 里面调用
	app.Action = gptn      //默认的操作，就是启动一个gptn节点， 如果有其他子命令行参数，会调用到下面的Commands里面去
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2017-2018 The go-palletone Authors"
	// 设置各个子命令的处理类/函数，比如consoleCommand 最后调用到 localConsole
	// 如果命令行参数里面有下面的指令，就会直接调用下面的Command.Run方法，而不调用默认的app.Action方法
	// Commands 是所有支持的子命令
	app.Commands = []cli.Command{
		// See chaincmd.go:
		initCommand, //初始化创世单元命令
		//importCommand,
		//exportCommand,
		//importPreimagesCommand,
		//exportPreimagesCommand,
		copydbCommand,
		removedbCommand,
		//dumpCommand,	//转储命令
		// See monitorcmd.go:
		monitorCommand,
		// See accountcmd.go:
		accountCommand,
		// walletCommand,
		// See consolecmd.go:
		consoleCommand, //js控制台命令
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		makecacheCommand,
		makedagCommand,
		versionCommand,
		bugCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand, //转储配置命令
		createGenesisJsonCommand,	// 创建创世json文件命令
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	// 所有能够解析的Options
	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)

	//before函数在app.Run的开始会先调用，也就是gopkg.in/urfave/cli.v1/app.go Run函数的前面
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		// Start system runtime metrics collection
		go metrics.CollectProcessMetrics(3 * time.Second)

		utils.SetupNetwork(ctx)
		return nil
	}

	//after函数在最后调用，app.Run 里面会设置defer function
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	// 如果是gptn命令行启动，不带子命令，那么直接调用app.Action = gptn()函数；
	// 如果带有子命令比如gptn console，那么会调用Command.Run, 最终会执行该子命令对应的Command.Action
	// 对于console子命令来说就是调用的 localConsole()函数；
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// gptn is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
// 默认情况下，如果不带子命令参数，那么app.Action = gptn，也就会调用gptn()函数 来启动PalletOne
func gptn(ctx *cli.Context) error {
	// 根据参数来新建一个全节点服务
	node := makeFullNode(ctx)

	// 创建协程启动节点，然后进入等待状态(阻塞模式)。
	// 启动 serviceFuncs 列表中的所有匿名服务，在Node.Start()中执行，函数调用路径为：
	/*
		1. startNode(ctx, node);
		2. utils.StartNode(stack);
		3. stack.Start() ;
	*/
	startNode(ctx, node)
	node.Wait()
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node) {
	debug.Memsize.Add("node", stack)

	// Start up the node itself
	utils.StartNode(stack)

	// Unlock any account specifically requested
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	//自动解锁指定的账号，配置的, 这样非交互状态下方便使用
	passwords := utils.MakePasswordList(ctx)
	unlocks := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for i, account := range unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, ks, trimmed, i, passwords)
		}
	}

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	// 创建协程，用RPC监听钱包创建事件。
	go func() {
		// Create a chain state reader for self-derivation
		rpcClient, err := stack.Attach()
		if err != nil {
			utils.Fatalf("Failed to attach to self: %v", err)
		}
		stateReader := ptnclient.NewClient(rpcClient)

		// Open any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				if event.Wallet.URL().Scheme == "ledger" {
					event.Wallet.SelfDerive(accounts.DefaultLedgerBaseDerivationPath, stateReader)
				} else {
					event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
				}

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
	// Start auxiliary services if enabled
	//如果指定了--mine 选项，就自动开始挖矿
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) || ctx.GlobalBool(utils.DeveloperFlag.Name) {
		// Mining only makes sense if a full PalletOne node is running
		if ctx.GlobalBool(utils.LightModeFlag.Name) || ctx.GlobalString(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support mining")
		}
		var ethereum *ptn.PalletOne
		if err := stack.Service(&ethereum); err != nil {
			utils.Fatalf("PalletOne service not running: %v", err)
		}

		// Use a reduced number of threads if requested
		if threads := ctx.GlobalInt(utils.MinerThreadsFlag.Name); threads > 0 {
			type threaded interface {
				SetThreads(threads int)
			}
			//if th, ok := ethereum.Engine().(threaded); ok {
			//	th.SetThreads(threads)
			//}
		}
		// Set the gas price to the limits from the CLI
		ethereum.TxPool().SetGasPrice(utils.GlobalBig(ctx, utils.GasPriceFlag.Name))
		//开启挖矿，创建协程到后台处理
		//if err := ethereum.StartMining(true); err != nil {
		//	utils.Fatalf("Failed to start mining: %v", err)
		//}
	}

}
