package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	count = flag.Int("c", 0, "Number of pings to send. If count is 0 or negative, will ping forever.")
	debug = flag.Bool("debug", false, "Show debug info.")

	data = []byte("FOO")
)

func main() {
	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatal("Must provide target host.")
	}
	host := flag.Args()[0]

	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatalf("Failed to look up host IP: %v", err)
	}
	if len(ips) == 0 {
		log.Fatalf("Got no IPs for host: %v", host)
	}

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Printf("PING %s (%s): %d data bytes\n", host, ips[0].String(), len(data))

	for seq := 0; *count <= 0 || seq < *count; seq++ {
		wm := &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  seq,
				Data: data,
			},
		}
		wb, err := wm.Marshal(nil)
		if err != nil {
			log.Fatal(err)
		}

		start := time.Now()
		if _, err = conn.WriteTo(wb, &net.UDPAddr{IP: ips[0]}); err != nil {
			log.Fatal(err)
		}

		rb := make([]byte, 1500)
		n, dst, err := conn.ReadFrom(rb)
		if err != nil {
			log.Fatal(err)
		}
		elapsed := time.Since(start)

		rm, err := icmp.ParseMessage(1, rb[:n])
		if err != nil {
			log.Fatal(err)
		}

		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			b := rm.Body.(*icmp.Echo)
			fmt.Printf("%d bytes from %s: seq=%d time=%.3f ms\n", n, strings.Split(dst.String(), ":")[0], b.Seq, elapsed.Seconds()*1000)

			if *debug {
				log.Printf("%+v", b)
				log.Printf("Data: %q", b.Data)
			}
		default:
			if *debug {
				log.Printf("Got something else: %+v", rm)
			}
		}
	}
}
