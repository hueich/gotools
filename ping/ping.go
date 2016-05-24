package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	count   = flag.Int("c", 0, "Number of pings to send. If count is 0 or negative, will ping forever.")
	version = flag.Int("v", 4, "Version of IP address to use when looking up the hostname, either 4 or 6 for IPv4 or IPv6, respectively. Other values not supported.")
	debug   = flag.Bool("debug", false, "Show debug info.")

	data = []byte("FOO")
)

func main() {
	flag.Parse()

	switch *version {
	case ipv4.Version:
	case ipv6.Version:
		break
	default:
		log.Fatal("IP version must be either 4 or 6.")
	}

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
	log.Printf("IPs: %v", ips)

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

func pickIP(ips []net.IP, ver int) net.IP {
	for _, ip := range ips {
		isV4 := ip.To4() != nil
		if ver == ipv4.Version && isV4 || ver == ipv6.Version && !isV4 {
			return ip
		}
	}
	return nil
}