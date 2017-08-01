package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var counter uint

	for {
		select {
		case <-time.After(time.Second):
			fmt.Println(counter, "Eating some glue with PID:", os.Getpid())
		case <-sigs:
			os.Exit(0)
		}
		counter++
	}
}
