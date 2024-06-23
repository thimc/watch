# Watch

Watch runs a command each time a set of files changes. Watch is a
clone of the [9fans/go Watch
tool](https://github.com/9fans/go/blob/main/acme/Watch/main.go) but
is made to work outside of Acme.

There are some differences between the 9fans/go Watch tool and this
tool, instead of relying on the current working directory to find
files to monitor, the user needs to specify (with globbing support)
which files to specifically watch. Another difference is the notion
of an unescaped `%` in the arguments list which is substituted with
file path of the file that was changed.

Watch may not be as efficient as it could be (using syscalls) but is
written so that it is as simple and cross-platform as possible.

## Usage

    watch '*.go' go run \%

## License

MIT
