package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/jlevesy/stracebeat/probe"
)

func dump(sample probe.Sample) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 8, ' ', tabwriter.AlignRight|tabwriter.Debug)

	for k, v := range sample {
		if v > 0 {
			fmt.Fprintf(w, "%d\t%d\n", k, v)
		}
	}

	w.Flush()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("You must give 1 pid")
		os.Exit(1)
	}

	pid, err := strconv.Atoi(os.Args[1])

	if err != nil {
		fmt.Println("Failed to parse pid: ", err)
		os.Exit(1)
	}

	p := probe.New(1)
	defer p.Stop()

	err = p.Attach(pid)

	if err != nil {
		fmt.Println("Failed to attach to pid: ", err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-time.After(time.Second):
			sample, err := p.Collect()
			if err != nil {
				fmt.Println("Watch reported error: ", err)
				os.Exit(1)
			}
			dump(sample)
		case <-sigs:
			break
		}
	}
}
