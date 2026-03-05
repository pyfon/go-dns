package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel) // FIXME: set to info.
}

func main() {
	var sockets SocketList
	zonePath := flag.String("zones", "", "A path to a directory containing one or more zone files")
	logLevel := flag.String("logLevel", "info", "log level (debug, info, warn, error, fatal, panic)")
	flag.Var(&sockets, "listen", "Listen on a given ADDR:PORT pair. (use flag multiple times for multiple sockets)")
	flag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Invalid logLevel given: %v", err)
		os.Exit(1)
	}
	log.SetLevel(level)

	if len(*zonePath) <= 0 {
		log.Errorf("Missing required argument: -zones")
		os.Exit(1)
	}
	log.Debugf("Parsing zone files in %s", *zonePath)

	zoneFilePaths, err := getZoneFilePaths(*zonePath)
	if err != nil {
		log.Errorf("Couldn't gather zone files in %v: %v", *zonePath, err)
		os.Exit(1)
	}

	zones, err := parseZoneFiles(zoneFilePaths)
	if err != nil {
		log.Errorf("Could not parse zone files: %v", err)
		os.Exit(1)
	}

	var g errgroup.Group
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

// getFiles returns a list of valid, resolved file paths of all files recursively found under dirPath.
func getZoneFilePaths(zoneDirPath string) ([]string, error) {
	var files []string

	evalDirEnt := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		absPath, err := filepath.Abs(realPath)
		if err != nil {
			return err
		}
		files = append(files, absPath)
		return nil
	}

	if err := filepath.WalkDir(zoneDirPath, evalDirEnt); err != nil {
		return nil, err
	}

	return files, nil
}

// parseZoneFiles takes a list of zone file paths, parses each one into a Zone object,
// and returns a trie of Zones for fast lookup.
func parseZoneFiles(zoneFiles []string) (ZoneTrie, error) {
	var zones map[Domain]Zone = make(map[Domain]Zone)
	for _, file := range zoneFiles {
		zoneFile, err := os.Open(file)
		if err != nil {
			return ZoneTrie{}, err
		}
		zoneReader := bufio.NewReader(zoneFile)
		lexer := NewLexer(zoneReader)
		parser := NewParser(&lexer, filepath.Base(file))
		zone, err := parser.Parse()
		zoneFile.Close()
		if err != nil {
			return ZoneTrie{}, err
		}
		if _, exists := zones[zone.Name]; exists {
			errStr := fmt.Sprintf("Duplicate zone: %v", zone.Name)
			return ZoneTrie{}, errors.New(errStr)
		}
		zones[zone.Name] = zone
	}
	return NewZoneTrie(zones), nil
}
