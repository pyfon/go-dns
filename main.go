package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel) // FIXME: set to info.
}

func main() {
	log.Info("Nathan's DNS Server")

	if len(os.Args) <= 1 {
		log.Error("Missing command line argument: path to a directory of zone files")
		os.Exit(1)
	}

	zonePath := os.Args[1]
	log.Debugf("Parsing zone files in %s", zonePath)

	files, err := getFiles(zonePath)
	if err != nil {
		log.Errorf("Couldn't gather zone files in %v: %v", zonePath, err)
		os.Exit(1)
	}

	var zones map[Domain]*Zone = make(map[Domain]*Zone) // Indexed by zone name (domain)
	for _, file := range files {
		zoneFile, err := os.Open(file)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		zoneReader := bufio.NewReader(zoneFile)
		lexer := NewLexer(zoneReader)
		parser := NewParser(&lexer, filepath.Base(file))
		zone, err := parser.Parse()
		zoneFile.Close()
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		if _, exists := zones[zone.Zone]; exists {
			log.Errorf("Duplicate zone: %v", zone.Zone.String())
			os.Exit(1)
		}
		zones[zone.Zone] = &zone
	}

	// --------- DEBUG REMOVE FIXME ---------
	for _, zone := range zones {
		fmt.Printf("%v\n", zone)
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Best match:\n%v\n", FindBestZoneMatch(zones, Domain(line)))
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
	}
	// --------- DEBUG REMOVE FIXME ---------
}

// getFiles returns a list of valid, resolved file paths of all files recursively found under dirPath.
func getFiles(dirPath string) ([]string, error) {
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

	if err := filepath.WalkDir(dirPath, evalDirEnt); err != nil {
		return nil, err
	}

	return files, nil
}
