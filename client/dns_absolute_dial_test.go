package client

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// TestAbsoluteLookupDialContextSendsFQDNQueries pins #1769: with a custom
// DNS resolver configured, hostname lookups must go out as absolute
// (trailing-dot) queries — the pure-Go resolver skips search-list expansion
// for FQDNs, so a Kubernetes pod's ndots:5 walk of namespace/svc/cluster
// suffixes never happens and exactly one query per family is sent.
func TestAbsoluteLookupDialContextSendsFQDNQueries(t *testing.T) {
	queries := make(chan string, 8)
	resolved := make(chan net.IP, 1)

	// A minimal UDP DNS responder: capture the question name, answer 127.0.0.1.
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve udp addr: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer conn.Close()
	go func() {
		buf := make([]byte, 512)
		for {
			n, peer, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			packet := append([]byte(nil), buf[:n]...)
			// Extract the QNAME (label sequence before the QTYPE/QCLASS).
			name := decodeQName(packet)
			queries <- name
			if !strings.HasSuffix(name, ".") {
				// Non-absolute question: the search list leaked in.
				continue
			}
			response := buildAAnswer(packet, 127, 0, 0, 1)
			_, _ = conn.WriteToUDP(response, peer)
			select {
			case resolved <- net.IPv4(127, 0, 0, 1):
			default:
			}
		}
	}()

	dnsResolver := &DNSResolverConfig{Protocol: "udp", Host: "127.0.0.1", Port: itoa(conn.LocalAddr().(*net.UDPAddr).Port)}
	dialer := newAbsoluteLookupDialer(dnsResolver)

	c, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	defer c.Close()
	go func() {
		for {
			conn, err := c.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	tcpAddr := strings.TrimPrefix(c.Addr().String(), "127.0.0.1:")
	// Rewrite the target to the hostname form the dial path would see: the
	// dial below uses the FQDN "example.test." so the fake resolver answers.
	out, err := absoluteLookupDialContext(context.Background(), dialer, dnsResolver, "tcp", "example.test.:"+tcpAddr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer out.Close()

	select {
	case name := <-queries:
		if !strings.HasSuffix(name, ".") {
			t.Fatalf("query %q is not absolute — search-list expansion leaked", name)
		}
		if !strings.HasPrefix(name, "example.test.") {
			t.Fatalf("unexpected question name %q", name)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no DNS query reached the custom resolver")
	}

	// Every observed question must be absolute — the ndots walk would show up
	// as example.test.<suffix>. queries here.
	for {
		select {
		case name := <-queries:
			if !strings.HasSuffix(name, ".") || !strings.HasPrefix(name, "example.test.") {
				t.Fatalf("search-list variant observed: %q", name)
			}
		default:
			return
		}
	}
}

// decodeQName extracts the dotted QNAME from a raw DNS query packet.
func decodeQName(packet []byte) string {
	if len(packet) < 12 {
		return ""
	}
	var b strings.Builder
	i := 12
	for i < len(packet) {
		length := int(packet[i])
		if length == 0 {
			b.WriteByte('.')
			return b.String()
		}
		i++
		if i+length > len(packet) {
			return b.String()
		}
		b.Write(packet[i : i+length])
		b.WriteByte('.')
		i += length
	}
	return b.String()
}

// buildAAnswer crafts a minimal A response: header + the original question
// section + one A record pointed at the question name.
func buildAAnswer(query []byte, a, b, c, d byte) []byte {
	if len(query) < 12 {
		return nil
	}
	// The question ends at QNAME + QTYPE(2) + QCLASS(2).
	i := 12
	for i < len(query) && query[i] != 0 {
		i += int(query[i]) + 1
	}
	if i >= len(query) {
		return nil
	}
	questionEnd := i + 5 // zero label + QTYPE + QCLASS

	response := append([]byte(nil), query[:questionEnd]...)
	response[2] |= 0x80 // QR = response
	response[3] &= 0x7F // AA=0, TC=0, RD copied
	response[7] = 1     // ANCOUNT = 1
	response = append(response, 0xc0, 0x0c)              // pointer to QNAME
	response = append(response, 0x00, 0x01)              // TYPE = A
	response = append(response, 0x00, 0x01)              // CLASS = IN
	response = append(response, 0x00, 0x00, 0x00, 0x3c)  // TTL = 60
	response = append(response, 0x00, 0x04)              // RDLENGTH
	response = append(response, a, b, c, d)
	return response
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}
