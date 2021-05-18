package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jrdnull/cron"
)

func main() {
	exp, err := cron.Parse(strings.Join(os.Args[1:], " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	fmt.Print(exp)
}
