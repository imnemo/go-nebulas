// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/nebulasio/go-nebulas/neblet"
	"github.com/nebulasio/go-nebulas/util/logging"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// const
const (
	PanicLevel = "panic"
	FatalLevel = "fatal"
	ErrorLevel = "error"
	WarnLevel  = "warn"
	InfoLevel  = "info"
	DebugLevel = "debug"
)

var (
	version   string
	commit    string
	branch    string
	compileAt string
	config    string
)

func main() {

	app := cli.NewApp()
	app.Action = neb
	app.Name = "neb"
	app.Version = fmt.Sprintf("%s, branch %s, commit %s", version, branch, commit)
	timestamp, _ := strconv.ParseInt(compileAt, 10, 64)
	app.Compiled = time.Unix(timestamp, 0)
	app.Usage = "the go-nebulas command line interface"
	app.Copyright = "Copyright 2017-2018 The go-nebulas Authors"

	app.Flags = append(app.Flags, ConfigFlag)
	app.Flags = append(app.Flags, NetworkFlags...)
	app.Flags = append(app.Flags, ChainFlags...)
	app.Flags = append(app.Flags, RPCFlags...)
	app.Flags = append(app.Flags, StatsFlags...)

	sort.Sort(cli.FlagsByName(app.Flags))

	app.Commands = []cli.Command{
		initCommand,
		genesisCommand,
		accountCommand,
		consoleCommand,
		networkCommand,
		versionCommand,
		licenseCommand,
		configCommand,
		blockDumpCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Run(os.Args)
}

func neb(ctx *cli.Context) error {
	logging.EnableFuncNameLogger()

	n, err := makeNeb(ctx)
	if err != nil {
		return err
	}

	if n.Config().App.EnableCrashReport {
		InitCrashReporter()
	}

	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
	if n.Config().App.LogFileEnable && len(n.Config().App.LogFileDir) > 0 {
		fileHook, err := logging.NewLogrusFileHook(n.Config().App.LogFileDir)
		if err != nil {
			panic("Setup Neblet Failed: " + err.Error())
		}
		log.AddHook(fileHook)
	}

	if n.Config().App.LogLevel != "" {
		switch n.Config().App.LogLevel {
		case PanicLevel:
			log.SetLevel(log.PanicLevel)
		case FatalLevel:
			log.SetLevel(log.FatalLevel)
		case ErrorLevel:
			log.SetLevel(log.ErrorLevel)
		case WarnLevel:
			log.SetLevel(log.WarnLevel)
		case InfoLevel:
			log.SetLevel(log.InfoLevel)
		case DebugLevel:
			log.SetLevel(log.DebugLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	}

	runNeb(n)

	// TODO: just use the signal to block main.
	for {
		time.Sleep(60 * time.Second) // or runtime.Gosched() or similar per @misterbee
	}
}

func runNeb(n *neblet.Neblet) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if err := n.Setup(); err != nil {
		panic("Setup Neblet Failed: " + err.Error())
	}

	if err := n.Start(); err != nil {
		panic("Start Neblet Failed: " + err.Error())
	}

	go func() {
		<-c
		n.Stop()

		// TODO: remove this once p2pManager handles stop properly.
		os.Exit(1)
	}()
}

func makeNeb(ctx *cli.Context) (*neblet.Neblet, error) {
	conf := neblet.LoadConfig(config)

	// load config from cli args
	networkConfig(ctx, conf.Network)
	chainConfig(ctx, conf.Chain)
	rpcConfig(ctx, conf.Rpc)
	statsConfig(ctx, conf.Stats)

	n, err := neblet.New(*conf)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// FatalF fatal format err
func FatalF(format string, args ...interface{}) {
	err := fmt.Sprintf(format, args...)
	fmt.Println(err)
	os.Exit(1)
}
