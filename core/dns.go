package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

var (
	// ErrDNSWithNoQueryName
	ErrDNSWithNoQueryName = errors.New("you must specify query name for DNS")
	// ErrDNSWithInvalidQueryType
	ErrDNSWithInvalidQueryType = errors.New("invalid query type")
)

const (
	dnsPort = 53
)

type DNS struct {
	// QueryType is the type for the DNS records like A,AAAA, CNAME...
	QueryType string `yaml:"query-type"`
	// QueryName is the query for DNS
	QueryName string `yaml:"query-name"`
}

func (d *DNS) validateAndSetDefault() {
	if len(d.QueryName) == 0 {
		panic(ErrDNSWithNoQueryName)
	}

	if !strings.HasSuffix(d.QueryName, ".") {
		d.QueryName += "."
	}
	if _, ok := dns.StringToType[d.QueryType]; !ok {
		panic(ErrDNSWithInvalidQueryType)
	}
}

func (d *DNS) query(url string, result *Result) {
	if !strings.Contains(url, ":") {
		url = fmt.Sprintf("%s:%d", url, dnsPort)
	}
	queryType := dns.StringToType[d.QueryType]
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(d.QueryName, queryType)
	r, _, err := c.Exchange(m, url)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return
	}
	result.Connected = true
	result.DNSRCode = dns.RcodeToString[r.Rcode]
	for _, rr := range r.Answer {
		switch rr.Header().Rrtype {
		case dns.TypeA:
			if a, ok := rr.(*dns.A); ok {
				result.Body = []byte(a.A.String())
			}
		case dns.TypeAAAA:
			if aaaa, ok := rr.(*dns.AAAA); ok {
				result.Body = []byte(aaaa.AAAA.String())
			}
		case dns.TypeCNAME:
			if cname, ok := rr.(*dns.CNAME); ok {
				result.Body = []byte(cname.Target)
			}
		case dns.TypeMX:
			if mx, ok := rr.(*dns.MX); ok {
				result.Body = []byte(mx.Mx)
			}
		case dns.TypeNS:
			if ns, ok := rr.(*dns.NS); ok {
				result.Body = []byte(ns.Ns)
			}
		default:
			result.Body = []byte("not supported")
		}
	}
}
