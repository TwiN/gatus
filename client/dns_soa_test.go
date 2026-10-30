package client

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestQueryDNS_SOA(t *testing.T) {
	t.Parallel()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(req)
		m.Answer = append(m.Answer, &dns.SOA{
			Hdr:     dns.RR_Header{Name: req.Question[0].Name, Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 3600},
			Ns:      "ns1.example.com.",
			Mbox:    "hostmaster.example.com.",
			Serial:  2026103101,
			Refresh: 7200,
			Retry:   3600,
			Expire:  1209600,
			Minttl:  300,
		})
		_ = w.WriteMsg(m)
	})
	server := &dns.Server{PacketConn: pc, Handler: handler}
	go func() { _ = server.ActivateAndServe() }()
	defer server.Shutdown()

	connected, dnsRcode, body, err := QueryDNS("SOA", "example.com.", pc.LocalAddr().String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !connected {
		t.Error("expected connected to be true")
	}
	if dnsRcode != "NOERROR" {
		t.Errorf("expected DNSRCode to be NOERROR, got %s", dnsRcode)
	}
	expectedBody := "ns1.example.com. hostmaster.example.com. 2026103101 7200 3600 1209600 300"
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}
