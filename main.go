// Command watch runs a command each time a set of files changes.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type file struct {
	Path string
	Cmd  []string
	Stat os.FileInfo
}

var (
	dur     = flag.Int("d", 1, "delay (in seconds) between each poll")
	verbose = flag.Bool("v", false, "verbose")
)

func watch(f file, ch chan file) {
	var dur = time.Second * time.Duration(*dur)
	for {
		time.Sleep(dur)
		s, err := os.Stat(f.Path)
		if err != nil {
			continue
		}
		if s.Size() != f.Stat.Size() || s.Mode() != f.Stat.Mode() || s.ModTime() != f.Stat.ModTime() {
			f.Stat = s
			if *verbose {
				log.Printf("%s changed\n", f.Path)
			}
			ch <- f
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-v] [-d delay] pattern cmd [args...]\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var workch = make(chan file)
	flag.Usage = usage
	flag.Parse()
	var (
		args    = flag.Args()
		matches []string
		err     error
	)
	if fi, _ := os.Stdin.Stat(); (fi.Mode() & os.ModeCharDevice) == 0 {
		r := bufio.NewReader(os.Stdin)
		for {
			ln, err := r.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			matches = append(matches, strings.Trim(ln, "\n"))
		}
	} else {
		if len(args) < 2 {
			usage()
		}
		matches, err = filepath.Glob(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q: %s\n", args[0], err)
			os.Exit(1)
		}
		args = args[1:]
	}
	if *verbose {
		log.Printf("watching: %+v\n", matches)
	}
	if len(matches) < 1 {
		fmt.Fprintf(os.Stderr, "could not match file pattern: %q\n", args[0])
		os.Exit(1)
	}
	for _, f := range matches {
		stat, err := os.Stat(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stat %q: %s\n", f, err)
			continue
		}
		var cmd []string
		for _, arg := range args {
			if arg == "%" {
				arg = f
			} else if strings.Contains(arg, "\\%") {
				arg = strings.Replace(arg, "\\%", "%", -1)
			}
			cmd = append(cmd, arg)
		}
		go watch(file{
			Path: f,
			Cmd:  cmd,
			Stat: stat,
		}, workch)
	}

	for file := range workch {
		if *verbose {
			log.Printf("running %q with args %+v\n", file.Cmd[0], file.Cmd[1:])
		}
		e := exec.Command(file.Cmd[0], file.Cmd[1:]...)
		e.Stdout = os.Stdout
		e.Stderr = os.Stderr
		e.Stdin = os.Stdin
		if err := e.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "run: %s\n", err)
		}
	}
}
