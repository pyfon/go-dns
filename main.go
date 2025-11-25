package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel) // FIXME: set to info.
}

func main() {
	var hosts HostList
	var ports PortList
	zonePath := flag.String("zones", "", "A path to a directory containing one or more zone files")
	logLevel := flag.String("logLevel", "info", "log level (debug, info, warn, error, fatal, panic)")
	flag.Var(&hosts, "addr", "Listen on given address (use flag multiple times for multiple addresses)")
	flag.Var(&ports, "port", "Listen on given port (use flag multiple times for multiple ports)")
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

	fmt.Printf("%v\n", zones) // TODO REMOVE
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
// and returns a map of pointers to corresponding zone objects, indexed by zone name.
func parseZoneFiles(zoneFiles []string) (map[Domain]*Zone, error) {
	var zones map[Domain]*Zone = make(map[Domain]*Zone)
	for _, file := range zoneFiles {
		zoneFile, err := os.Open(file)
		if err != nil {
			return zones, err
		}
		zoneReader := bufio.NewReader(zoneFile)
		lexer := NewLexer(zoneReader)
		parser := NewParser(&lexer, filepath.Base(file))
		zone, err := parser.Parse()
		zoneFile.Close()
		if err != nil {
			return zones, err
		}
		if _, exists := zones[zone.Name]; exists {
			errStr := fmt.Sprintf("Duplicate zone: %v", zone.Name)
			return zones, errors.New(errStr)
		}
		zones[zone.Name] = &zone
	}
	return zones, nil
}
