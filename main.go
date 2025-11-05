package main

import (
	"bufio"
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

	var zones []Zone = make([]Zone, 0, len(files))
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
		zones = append(zones, zone)
	}
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
