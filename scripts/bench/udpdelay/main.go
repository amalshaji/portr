// Command udpdelay relays UDP between clients and one upstream, adding fixed
// latency in each direction and an optional upload rate cap. It shapes the
// QUIC tunnel leg the way toxiproxy shapes the websocket one, since toxiproxy
// only relays TCP.
//
// Packets beyond the rate cap queue like they would in a home router; once
// -queue bytes are waiting, new packets are dropped. SIGUSR1 toggles a full
// outage that drops every packet.
package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var blackhole atomic.Bool

type packet struct {
	data []byte
	due  time.Time
}

// pipe delays packets and, when rate is set, spaces them to that many bytes
// per second before the latency is added.
type pipe struct {
	latency   time.Duration
	rate      float64
	queueMax  int
	mu        sync.Mutex
	departure time.Time
	queued    int
	out       chan packet
}

func newPipe(latency time.Duration, kbps, queueMax int, write func([]byte)) *pipe {
	p := &pipe{latency: latency, rate: float64(kbps) * 1000, queueMax: queueMax, out: make(chan packet, 65536)}
	go func() {
		for pkt := range p.out {
			time.Sleep(time.Until(pkt.due))
			p.mu.Lock()
			p.queued -= len(pkt.data)
			p.mu.Unlock()
			write(pkt.data)
		}
	}()
	return p
}

func (p *pipe) send(data []byte) {
	if blackhole.Load() {
		return
	}
	now := time.Now()
	p.mu.Lock()
	if p.queueMax > 0 && p.queued+len(data) > p.queueMax {
		p.mu.Unlock()
		return
	}
	departure := now
	if p.rate > 0 {
		if p.departure.After(departure) {
			departure = p.departure
		}
		departure = departure.Add(time.Duration(float64(len(data)) / p.rate * float64(time.Second)))
		p.departure = departure
	}
	p.queued += len(data)
	p.mu.Unlock()
	select {
	case p.out <- packet{data: data, due: departure.Add(p.latency)}:
	default:
		p.mu.Lock()
		p.queued -= len(data)
		p.mu.Unlock()
	}
}

func main() {
	listen := flag.String("listen", "127.0.0.1:14433", "address clients send to")
	upstream := flag.String("upstream", "127.0.0.1:4433", "address packets are relayed to")
	latency := flag.Duration("latency", 40*time.Millisecond, "delay in each direction")
	uploadKbps := flag.Int("upload-kbps", 0, "client to upstream cap in KB/s, 0 for none")
	queue := flag.Int("queue", 256<<10, "bytes queued behind the upload cap before dropping")
	flag.Parse()

	upstreamAddr, err := net.ResolveUDPAddr("udp", *upstream)
	if err != nil {
		log.Fatal(err)
	}
	listenAddr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	front, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)
	go func() {
		for range signals {
			log.Printf("blackhole=%v", !blackhole.Load())
			blackhole.Store(!blackhole.Load())
		}
	}()

	type session struct {
		conn *net.UDPConn
		up   *pipe
	}
	sessions := map[string]*session{}
	buf := make([]byte, 65536)
	for {
		n, client, err := front.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		data := append([]byte(nil), buf[:n]...)
		sess, ok := sessions[client.String()]
		if !ok {
			conn, err := net.DialUDP("udp", nil, upstreamAddr)
			if err != nil {
				log.Print(err)
				continue
			}
			clientAddr := client
			down := newPipe(*latency, 0, 0, func(b []byte) { _, _ = front.WriteToUDP(b, clientAddr) })
			sess = &session{conn: conn, up: newPipe(*latency, *uploadKbps, *queue, func(b []byte) { _, _ = conn.Write(b) })}
			sessions[client.String()] = sess
			go func() {
				reply := make([]byte, 65536)
				for {
					n, err := conn.Read(reply)
					if err != nil {
						return
					}
					down.send(append([]byte(nil), reply[:n]...))
				}
			}()
		}
		sess.up.send(data)
	}
}
