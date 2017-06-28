package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/net/icmp"
	"icmpEcho/ping"
)

func main() {

	p := ping.NewPinger()

	var mode bool
	switch len(os.Args) {
	case 1:
		mode = false
	case 2:
		mode = true
		err := p.SetAddr(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "    client mode: %s host\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "    server mode: %s\n", os.Args[0])
		os.Exit(1)
	}

	p.OnRecv = func(payload *icmp.Echo) {
		fmt.Println(string(payload.Data))
	}

	p.Run(mode)
}
