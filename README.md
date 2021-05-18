# cron

cron is package that parses and expands cron expressions (see the Godoc for
more on which features are supported).

Also provided is a command line which takes an expression as its only input
and prints the expanded expression.

## Usage

With a recent version of [Go installed](https://golang.org/doc/install).

You can build the command with `go build -o parse cmd/parse/main.go` or run it
directly, e.g:

```
% go run cmd/parse/main.go "*/15 0 1,15 * 1-5 /usr/bin/find"
minute		0 15 30 45
hour		0
day of month	1 15
month		1 2 3 4 5 6 7 8 9 10 11 12
day of week	1 2 3 4 5
command		/usr/bin/find
```

## Running tests

Tests can be run the same way as any other Go projects, e.g:

```
go test
```
