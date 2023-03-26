package core

import (
	"errors"
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

var (
	// ErrDNSWithNoQueryName is the error with which gatus will panic if a dns is configured without query name
	ErrDNSWithNoQueryName = errors.New("you must specify a query name for DNS")

	// ErrDNSWithInvalidQueryType is the error with which gatus will panic if a dns is configured with invalid query type
	ErrDNSWithInvalidQueryType = errors.New("invalid query type")
)

const (
	dnsPort = 53
)

// DNS is the configuration for a Endpoint of type DNS
type DNS struct {
	// QueryType is the type for the DNS records like A, AAAA, CNAME...
	QueryType string `yaml:"query-type"`

	// QueryName is the query for DNS
	QueryName string `yaml:"query-name"`
}

func (d *DNS) validateAndSetDefault() error {
	if len(d.QueryName) == 0 {
		return ErrDNSWithNoQueryName
	}
	if !strings.HasSuffix(d.QueryName, ".") {
		d.QueryName += "."
	}
	if _, ok := dns.StringToType[d.QueryType]; !ok {
		return ErrDNSWithInvalidQueryType
	}
	return nil
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
		result.AddError(err.Error())
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
			result.Body = []byte("query type is not supported yet")
		}
	}
}
