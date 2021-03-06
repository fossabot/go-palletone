## Go PalletOne

Official golang implementation of the palletone protocol.

[![Codacy Badge](https://api.codacy.com/project/badge/Grade/03e2a645bd5b40acabad69ff94833b02)](https://app.codacy.com/app/studyzy/go-palletone?utm_source=github.com&utm_medium=referral&utm_content=studyzy/go-palletone&utm_campaign=badger)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fstudyzy%2Fgo-palletone.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fstudyzy%2Fgo-palletone?ref=badge_shield)
 [![CircleCI](https://circleci.com/gh/palletone/go-palletone/tree/master.svg?style=shield)](https://circleci.com/gh/palletone/go-palletone/tree/master)
 [![Build Status](https://travis-ci.org/studyzy/go-palletone.svg?branch=master)](https://travis-ci.org/studyzy/go-palletone)

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/studyzy/go-palletone/wiki/Building-palletone)
on the wiki.

Building gptn requires both a Go (version 1.7 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make gptn

or, to build the full suite of utilities:

    make all

## Executables

The go-palletone project comes with several wrappers/executables found in the `cmd` directory.

| Command    | Description |
|:----------:|-------------|
| **`gptn`** | Our main palletone CLI client. It is the entry point into the palletone network (main-, test- or private net), capable of running as a full node (default) archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the palletone network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `gptn --help` and the [CLI Wiki page](https://github.com/studyzy/go-palletone/wiki/Command-Line-Options) for command line options. |

## Running gptn

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/studyzy/go-palletone/wiki/Command-Line-Options)), but we've
enumerated a few common parameter combos to get you up to speed quickly on how you can run your
own Gptn instance.

### Full node on the main palletone network

By far the most common scenario is people wanting to simply interact with the palletone network:
create accounts; transfer funds; deploy and interact with contracts. For this particular use-case
the user doesn't care about years-old historical data, so we can fast-sync quickly to the current
state of the network. To do so:

```
$ gptn --config /path/to/your_config.toml console 
```

This command will:

 * Start gptn in fast sync mode (default, can be changed with the `--syncmode` flag), causing it to
   download more data in exchange for avoiding processing the entire history of the palletone network,
   which is very CPU intensive.
 * Start up Gptn's built-in interactive [JavaScript console](https://github.com/studyzy/go-palletone/wiki/JavaScript-Console),
   (via the trailing `console` subcommand) through which you can invoke all official [`web3` methods](https://github.com/studyzy/wiki/wiki/JavaScript-API)
   as well as Gptn's own [management APIs](https://github.com/studyzy/go-palletone/wiki/Management-APIs).
   This too is optional and if you leave it out you can always attach to an already running Gptn instance
   with `gptn attach`.


### Configuration

As an alternative to passing the numerous flags to the `gptn` binary, you can also pass a configuration file via:

```
$ gptn --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to export your existing configuration:

```
$ gptn --your-favourite-flags dumpconfig
```

e.g. call it palletone.toml:

```
[Consensus]
Engine="solo"

[Log]
OutputPaths =["stdout","./log/all.log"]
ErrorOutputPaths= ["stderr","./log/error.log"]
LoggerLvl="info"   # ("debug", "info", "warn","error", "dpanic", "panic", and "fatal")
Encoding="console" # console,json
Development =true

[Dag]
DbPath="./leveldb"
DbName="palletone.db"

[Ada]
Ada1="ada1_config"
Ada2="ada2_config"

[Node]
DataDir = "./data1"
KeyStoreDir="./data1/keystore"
IPCPath = "./data1/gptn.ipc"
HTTPPort = 8541
HTTPVirtualHosts = ["0.0.0.0"]
HTTPCors = ["*"]

[Ptn]
NetworkId = 3369

[P2P]
ListenAddr = "0.0.0.0:30301"
#BootstrapNodes = ["pnode://228f7e50031457d804ce6021f4a211721bacb9abba9585870efea55780bb744005a7f22e22938040684cdec32c748968f5dbe19822d4fbb44c6aaa69e7abdfee@127.0.0.1:30301"]
```


### Operating a private network

Maintaining your own private network is more involved as a lot of configurations taken for granted in
the official networks need to be manually set up.

#### Defining the private genesis state

First, you'll need to create the genesis state of your networks, which all nodes need to be aware of and agree upon. This consists of a JSON file (e.g. call it `genesis.json`):

You can create a JSON file for the genesis state of a new chain with an existing account or a newly created account named `my-genesis.json` by running this command:

```
$ gptn create-genesis-json path/to/my-genesis.json
```

With the genesis state defined in the above JSON file, you'll need to initialize **every** Gptn node with it prior to starting it up to ensure all blockchain parameters are correctly set:

```
$ gptn init path/to/my-genesis.json
```

## Contribution

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to go-palletone, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit more
complex changes though, please check up with the core devs first on [our gitter channel](https://gitter.im/palletone/go-palletone)
to ensure those changes are in line with the general philosophy of the project and/or get some
early feedback which can make both your efforts much lighter as well as our review and merge
procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "ptn, rpc: make trace configs optional"

Please see the [Developers' Guide](https://github.com/studyzy/go-palletone/wiki/Developers'-Guide)
for more details on configuring your environment, managing project dependencies and testing procedures.

## License

The go-palletone binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fstudyzy%2Fgo-palletone.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fstudyzy%2Fgo-palletone?ref=badge_large)