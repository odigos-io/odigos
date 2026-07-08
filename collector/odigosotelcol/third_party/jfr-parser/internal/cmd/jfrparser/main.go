package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/grafana/jfr-parser/internal/cmd/jfrparser/format"
)

type command struct {
	// Opts
	format string

	// Args
	src  string
	dest string
}

func parseCommand(c *command) {
	c.format = "pprof"
	flag.Parse()
	args := flag.Args()
	c.src = args[0]
	if len(args) < 2 {
		c.dest = fmt.Sprintf("%s.%s", c.src, c.format)
	} else {
		c.dest = args[1]
	}
}

type formatter interface {
	// Formats the given JFR
	Format(buf []byte, dest string) ([]string, [][]byte, error)
}

// Usage: ./jfrparser [options] /path/to/jfr [/path/to/dest]
func main() {
	c := new(command)
	parseCommand(c)

	buf, err := os.ReadFile(c.src)
	if err != nil {
		panic(err)
	}

	var fmtr formatter = format.NewFormatterPprof()

	dests, data, err := fmtr.Format(buf, c.dest)
	if err != nil {
		panic(err)
	}

	if len(dests) != len(data) {
		panic(fmt.Errorf("logic error"))
	}

	for i := 0; i < len(dests); i++ {
		err = os.WriteFile(dests[i], data[i], 0644)
		if err != nil {
			panic(err)
		}
	}
}
