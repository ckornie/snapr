package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"snapr/internal/snapr"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var configuration string
var fileSystem string
var debug bool

var snap = &snapr.SnapArguments{}
var send = &snapr.SendArguments{}
var restore = &snapr.RestoreArguments{}

func init() {
	flag.BoolVar(&snap.Active, "snap", false, "Creates snapshots based on the configured file systems and intervals")
	flag.BoolVar(&send.Active, "send", false, "Sends new snapshots to the configured destinations")
	flag.BoolVar(&restore.Active, "restore", false, "Restores a file system from a bucket")
	flag.StringVar(&configuration, "configuration", "/etc/snapr.conf", "Specify an alternate configuration file")
	flag.StringVar(&fileSystem, "file-system", "", "A file system")
	flag.BoolVar(&debug, "debug", false, "Sets log level to debug")
}

func main() {
	flag.Parse()
	logger()

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := newContext()

	settings := snapr.NewSettings()
	err := settings.Load(configuration)
	if err != nil {
		return err
	}

	s, err := snapr.New(ctx, settings)
	if err != nil {
		return fmt.Errorf("unable to restore (%w)", err)
	}

	if snap.Active && !(restore.Active || send.Active) {
		s.Snap()
		return nil
	}

	if send.Active && !(restore.Active || snap.Active) {
		s.Send()
		return nil
	}

	if restore.Active && !(snap.Active || send.Active) {
		return runRestore(ctx, s)
	}

	return fmt.Errorf("invalid argument combination")
}

func runRestore(ctx context.Context, s *snapr.Snapr) error {
	if fileSystem == "" {
		return fmt.Errorf("no file system specified")
	}
	return s.Restore(fileSystem)
}

func logger() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	snapr.Logger = log.Logger

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("debug logging enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func newContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
		<-signals
		cancel()
	}()
	return ctx
}
