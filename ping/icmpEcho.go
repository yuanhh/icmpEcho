package ping

import (
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// parseICMPEcho parses b as an ICMP echo request or reply message body.
func ParseICMPEcho(b []byte) (*icmp.Echo, error) {
	bodylen := len(b)
	p := &icmp.Echo{ID: int(b[0])<<8 | int(b[1]), Seq: int(b[2])<<8 | int(b[3])}
	if bodylen > 4 {
		p.Data = make([]byte, bodylen-4)
		copy(p.Data, b[4:])
	}
	return p, nil
}

func SendICMPEcho(
	c *icmp.PacketConn,
	addr string,
	typ ipv4.ICMPType,
	bytes []byte,
) error {
	wm := icmp.Message{
		Type: typ,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: bytes,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}

	dst, err := net.ResolveIPAddr("ip4:icmp", addr)
	if _, err := c.WriteTo(wb, dst); err != nil {
		log.Fatal(err)
	}
	return err
}

func RecvICMPEcho(
	c *icmp.PacketConn,
	rb []byte,
	msec time.Duration,
) (int, net.Addr, error) {
	c.SetReadDeadline(time.Now().Add(time.Millisecond * msec))
	return c.ReadFrom(rb)
}
