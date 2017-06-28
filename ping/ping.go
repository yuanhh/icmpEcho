package ping

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	MTU          = 1500
	protocolICMP = 1
)

type Pinger struct {
	OnRecv func(*icmp.Echo)

	done chan bool

	ipaddr *net.IPAddr
	addr   string
	source string
}

type packet struct {
	bytes  []byte
	nbytes int
	peer   net.Addr
}

func NewPinger() *Pinger {
	return &Pinger{
		OnRecv: nil,
		ipaddr: nil,
		addr:   "",
		source: "0.0.0.0",
		done:   make(chan bool),
	}
}

func (p *Pinger) IPAddr() *net.IPAddr {
	return p.ipaddr
}

func (p *Pinger) Addr() string {
	return p.addr
}

func (p *Pinger) Source() string {
	return p.source
}

func (p *Pinger) SetIPAddr(ipaddr *net.IPAddr) {
	p.ipaddr = ipaddr
	p.addr = ipaddr.String()
}

func (p *Pinger) SetAddr(addr string) error {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return err
	}

	p.ipaddr = ipaddr
	p.addr = addr
	return nil
}

func (p *Pinger) SetSource(localaddr string) {
	p.source = localaddr
}

func (p *Pinger) Run(mode bool) {
	c, err := icmp.ListenPacket("ip4:icmp", p.source)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	recv := make(chan *packet, 5)
	wg.Add(1)
	go p.recvICMP(c, recv, &wg)
	if mode {
		fmt.Println("Client Mode")
		wg.Add(2)
		go p.sendICMP(c, &wg)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-sig:
			close(p.done)
		case <-p.done:
			wg.Wait()
			return
		case r := <-recv:
			err := p.processPacket(c, r)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (p *Pinger) recvICMP(
	c *icmp.PacketConn,
	recv chan<- *packet,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-p.done:
			return
		default:
			rb := make([]byte, MTU)
			n, peer, err := RecvICMPEcho(c, rb, 100)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						continue
					} else {
						return
					}
				}
			}
			recv <- &packet{bytes: rb, nbytes: n, peer: peer}
		}
	}
}

func (p *Pinger) sendICMP(c *icmp.PacketConn, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-p.done:
			return
		default:
			err := SendICMPEcho(c, p.addr, ipv4.ICMPTypeEcho, []byte("STREAMING_REQUEST"))
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Err == syscall.ENOBUFS {
						continue
					}
				}
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (p *Pinger) processPacket(c *icmp.PacketConn, recv *packet) error {

	rb := recv.bytes
	rm, err := icmp.ParseMessage(protocolICMP, rb[:recv.nbytes])
	if err != nil {
		return err
	}
	mb, err := rm.Body.Marshal(protocolICMP)
	if err != nil {
		return err
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		msg, _ := ParseICMPEcho(mb)
		handler := p.OnRecv
		if handler != nil {
			handler(msg)
		}
	case ipv4.ICMPTypeEcho:
		SendICMPEcho(c, recv.peer.String(), ipv4.ICMPTypeEchoReply, mb)
	default:
		log.Printf("got %+v\n", rm)
	}
	return nil
}
