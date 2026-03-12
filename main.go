package main

import (
	//"context"
	"errors"
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	sockets, zoneDirPath, err := parseArgs()
	if err != nil {
		log.Errorln(err)
		flag.Usage()
		os.Exit(1)
	}

	zones, err := parseZoneFiles(zoneDirPath)
	if err != nil {
		log.Errorf("Could not parse zone files: %v", err)
		os.Exit(1)
	}

	var g errgroup.Group
	//ctx := context.Background()
	for _, sock := range sockets {
		g.Go(func() error {
			return Serve(sock, zones)
		})
	}

	if err := g.Wait(); err != nil {
		log.Error(err)
		os.Exit(-1) // TODO use errgroup contexts to exit cleanly!
	}
}

func parseArgs() (sockets SocketList, zonePath string, err error) {
	flag.StringVar(&zonePath, "zones", "", "A path to a directory containing one or more zone files")
	logLevel := flag.String("logLevel", "info", "log level (debug, info, warn, error, fatal, panic)")
	flag.Var(&sockets, "listen", "Listen on a given ADDR:PORT pair. (use flag multiple times for multiple sockets)")
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		s := fmt.Sprintf("Invalid logLevel given: %v", err)
		err = errors.New(s)
		return
	}
	log.SetLevel(level)

	if len(zonePath) <= 0 {
		s := fmt.Sprintf("Missing required argument: -zones")
		err = errors.New(s)
		return
	}

	if len(sockets) <= 0 {
		s := fmt.Sprintf("Missing required argument: -listen")
		err = errors.New(s)
		return
	}

	return
}
