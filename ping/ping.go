package main

import (
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
)

func main() {
	host := os.Args[1]

	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	wm := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   1234,
			Seq:  1,
			Data: []byte("FOO"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatal(err)
	}
	if len(ips) == 0 {
		log.Fatal("Could not resolve IP for %v", host)
	}

	if _, err = c.WriteTo(wb, &net.UDPAddr{IP: ips[0], Port: 80}); err != nil {
		log.Fatal(err)
	}

	rb := make([]byte, 1500)
	n, dst, err := c.ReadFrom(rb)
	if err != nil {
		log.Fatal(err)
	}

	rm, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		log.Printf("Got reply from %v", dst)
		b := rm.Body.(*icmp.Echo)
		log.Printf("Data: %q", b.Data)
	default:
		log.Printf("Got something else: %+v", rm)
	}
}
