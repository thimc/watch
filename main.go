// Command watch runs a command when a set of files changes.
//
// Usage:
//
//	watch [ -d delay ] 'pattern' cmd [ args... ]
//
// Watch works by periodically checking the status of the specified
// files using [os.Stat]. When a change is detected (e.g., modification
// time or file size), the command specified by the user is executed.
//
// To save system resources, a delay is used (specified in seconds),
// which allows watch to intermittenly check for file changes.  This
// approach also prevents multiple rapid executions if several changes
// are made rapidly.
//
// The pattern (e.g., '*.go', '*' or a file name) specifies which files to
// watch. If the pattern is meant to be globbed it needs to be quoted to
// prevent the shell from interpreting it.  Watch only takes one pattern as
// an argument.
//
// The command to be executed needs to exist in `$PATH` or a full file
// path is required. If one or more unescaped `%` exists in the arguments
// list it will be replaced with the file path of the file that was
// modified.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	delay = flag.Int("d", 1, "delay (in seconds) between each poll")

	workch chan string

	mu    sync.Mutex
	files map[string]os.FileInfo
)

func watch(path string) {
	dur := time.Second * time.Duration(*delay)
	for {
		time.Sleep(dur)
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		mu.Lock()
		stat, ok := files[path]
		mu.Unlock()
		if ok && stat != nil {
			if fi.Size() == stat.Size() && fi.Mode() == stat.Mode() && fi.ModTime() == stat.ModTime() && fi.IsDir() == stat.IsDir() {
				continue
			}
		}
		workch <- path
		mu.Lock()
		files[path] = fi
		mu.Unlock()
	}
}

func run(path string, args []string) {
	var (
		cmdlist = strings.Join(args, " ")
		sb      strings.Builder
	)
	for i, r := range cmdlist {
		if i > 0 {
			if r == '%' && cmdlist[i-1] != '\\' {
				sb.WriteString(path)
			} else {
				sb.WriteRune(r)
			}
			continue
		}
		sb.WriteRune(r)
	}
	cmdargs := strings.Fields(sb.String())
	cmd := exec.Command(cmdargs[0], cmdargs[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Run()
}

func main() {
	workch = make(chan string)
	files = make(map[string]os.FileInfo)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [ -d delay ] 'pattern' cmd [ args... ]\n", os.Args[0])
	}
	flag.Parse()
	args := flag.Args()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	pattern := args[0]
	args = args[1:]
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	if len(matches) < 1 {
		matches = append(matches, pattern)
	}
	for _, path := range matches {
		fi, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		mu.Lock()
		files[path] = fi
		mu.Unlock()
		go watch(path)
	}
	for path := range workch {
		run(path, args)
	}
}
