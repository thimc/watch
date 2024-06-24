// Command watch runs a command each time a set of file changes.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type metaData struct {
	Cmd  []string
	Stat os.FileInfo
	C    chan struct{}
}

var durationFlag = flag.Int("d", 1, "delay (in seconds) between each poll")

func watch(filePath string, m metaData) {
	var dur = time.Second * time.Duration(*durationFlag)
	for {
		time.Sleep(dur)
		s, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if s.Size() != m.Stat.Size() ||
			s.Mode() != m.Stat.Mode() ||
			s.ModTime() != m.Stat.ModTime() {
			m.Stat = s
			m.C <- struct{}{}
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s pattern cmd [args...]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var fileMap = make(map[string]metaData)
	flag.Usage = usage
	flag.Parse()
	var args = flag.Args()
	if len(args) < 2 {
		usage()
	}
	matches, err := filepath.Glob(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "filepath/Glob: %s\n", err)
		os.Exit(1)
	}
	if len(matches) < 1 {
		fmt.Fprintf(os.Stderr, "error: could not match file pattern: %q\n", args[0])
		os.Exit(1)
	}
	for _, file := range matches {
		stat, err := os.Stat(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "os/Stat: %q %s\n", file, err)
			continue
		}
		var cmd []string
		for _, arg := range args[1:] {
			if arg == "%" {
				arg = file
			} else if strings.Contains(arg, "\\%") {
				arg = strings.Replace(arg, "\\%", "%", -1)
			}
			cmd = append(cmd, arg)
		}
		fileMap[file] = metaData{
			Cmd:  cmd,
			Stat: stat,
			C:    make(chan struct{}),
		}
		go watch(file, fileMap[file])
	}

	for {
		for _, file := range fileMap {
			<-file.C
			e := exec.Command(file.Cmd[0], file.Cmd[1:]...)
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			e.Stdin = os.Stdin
			if err := e.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "exec/Command: %s\n", err)
			}
		}
	}
}
