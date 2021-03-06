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
	"time"
)

var (
	count    = flag.Int("c", 0, "Number of pings to send. If count is 0 or negative, will ping forever.")
	version  = flag.Int("v", 4, "Version of IP address to use when looking up the hostname, either 4 or 6 for IPv4 or IPv6, respectively. Other values not supported.")
	interval = flag.Duration("i", 1*time.Second, "Interval of time between pings. Specified as a decimal number followed by units of time, e.g. 1.5s, 200ms, etc.")
	debug    = flag.Bool("debug", false, "Show debug info.")

	data  = []byte("FOO")
	buf   = make([]byte, 1500)
	stats = make([]stat, 0)
)

type stat struct {
	Seq     int
	Lost    bool
	Elapsed time.Duration
}

func main() {
	flag.Parse()

	var proto string
	var zeroIP net.IP
	var echoType icmp.Type

	switch *version {
	case ipv4.Version:
		proto = "udp4"
		zeroIP = net.IPv4zero
		echoType = ipv4.ICMPTypeEcho
	case ipv6.Version:
		proto = "udp6"
		zeroIP = net.IPv6zero
		echoType = ipv6.ICMPTypeEchoRequest
	default:
		log.Fatal("IP version must be either 4 or 6.")
	}

	if len(flag.Args()) < 1 {
		log.Fatal("Must provide target host.")
	}
	host := flag.Args()[0]

	defer printStats(host, &stats)

	ips, err := net.LookupIP(host)
	if err != nil {
		log.Fatalf("Failed to look up host IP: %v", err)
	}
	if len(ips) == 0 {
		log.Fatalf("Got no IPs for host: %v", host)
	}
	if *debug {
		log.Printf("IPs: %v", ips)
	}

	ip, err := pickIP(ips, *version)
	if err != nil {
		log.Fatalf("Error picking host IP: %v", err)
	}

	conn, err := icmp.ListenPacket(proto, zeroIP.String())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	addr := &net.UDPAddr{IP: ip}

	fmt.Printf("PING %s (%s): %d data bytes\n", host, ip.String(), len(data))

	seq := 0
	// Send the first ping right away, instead of waiting for the first tick.
	if sendPing(conn, addr, echoType, data, &seq) {
		for _ = range time.Tick(*interval) {
			if !sendPing(conn, addr, echoType, data, &seq) {
				break
			}
		}
	}
}

// sendPing sends a ICMP echo message to the address. The function returns whether to continue sending pings.
func sendPing(conn *icmp.PacketConn, addr net.Addr, echoType icmp.Type, data []byte, seq *int) bool {
	wm := &icmp.Message{
		Type: echoType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  *seq,
			Data: data,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}
	s := stat{Seq: *seq}

	start := time.Now()
	if _, err = conn.WriteTo(wb, addr); err != nil {
		log.Fatal(err)
	}

	n, dest, err := conn.ReadFrom(buf)
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start)
	s.Elapsed = elapsed
	stats = append(stats, s)

	if *debug {
		log.Printf("dest: %s", dest.String())
	}

	rm, err := icmp.ParseMessage(1, buf[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		host, _, err := net.SplitHostPort(dest.String())
		if err != nil {
			log.Fatal(err)
		}
		b := rm.Body.(*icmp.Echo)
		fmt.Printf("%d bytes from %s: icmp_seq=%d time=%.3f ms\n", n, host, b.Seq, toMs(elapsed))

		if *debug {
			log.Printf("%+v", b)
			log.Printf("Data: %q", b.Data)
		}
	default:
		if *debug {
			log.Printf("Got something else: %+v", rm)
			break
		}
		fmt.Printf("Got a non-echo reply: %+v\n", rm)
	}

	*seq++
	return *count <= 0 || *seq < *count
}

func pickIP(ips []net.IP, ver int) (net.IP, error) {
	for _, ip := range ips {
		isV4 := ip.To4() != nil
		if ver == ipv4.Version && isV4 || ver == ipv6.Version && !isV4 {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("no available IPv%d address", ver)
}

func printStats(host string, stats *[]stat) error {
	fmt.Printf("\n--- %s ping statistics ---\n", host)
	nLost := 0
	var min, max, sum time.Duration
	for _, s := range *stats {
		sum += s.Elapsed
		if s.Elapsed > max {
			max = s.Elapsed
		}
		if min == 0 || s.Elapsed < min {
			min = s.Elapsed
		}
		if s.Lost {
			nLost++
		}
	}
	nSent := len(*stats)
	nGot := nSent - nLost
	lossRate := 0.0
	if nSent > 0 {
		lossRate = float64(nLost) / float64(nSent)
	}
	fmt.Printf("%d packets transmitted, %d packets received, %.1f%% packet loss\n", nSent, nGot, lossRate*100)
	avg := toMs(sum) / float64(nGot)
	fmt.Printf("round-trip min/avg/max = %.3f/%.3f/%.3f ms\n", toMs(min), avg, toMs(max))
	return nil
}
func toMs(d time.Duration) float64 {
	return d.Seconds() * 1000
}
