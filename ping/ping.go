package main

import (
	"flag"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
)

var (
	count = flag.Uint("c", 0, "Number of pings to send. If count is 0, will ping forever.")
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatal("Must provide target host.")
	}
	host := flag.Args()[0]

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for cnt := *count; *count == 0 || cnt > 0; cnt-- {
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
			log.Fatalf("Could not resolve IP for %v", host)
		}

		if _, err = conn.WriteTo(wb, &net.UDPAddr{IP: ips[0], Port: 80}); err != nil {
			log.Fatal(err)
		}

		rb := make([]byte, 1500)
		n, dst, err := conn.ReadFrom(rb)
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
}
