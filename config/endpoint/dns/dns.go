package dns

import (
	"errors"
	"strings"

	"github.com/miekg/dns"
)

var (
	// ErrDNSWithNoQueryName is the error with which gatus will panic if a dns is configured without query name
	ErrDNSWithNoQueryName = errors.New("you must specify a query name in the DNS configuration")

	// ErrDNSWithInvalidQueryType is the error with which gatus will panic if a dns is configured with invalid query type
	ErrDNSWithInvalidQueryType = errors.New("invalid query type in the DNS configuration")
)

// Config for an Endpoint of type DNS
type Config struct {
	// QueryType is the type for the DNS records like A, AAAA, CNAME...
	QueryType string `yaml:"query-type"`

	// QueryName is the query for DNS
	QueryName string `yaml:"query-name"`
}

func (d *Config) ValidateAndSetDefault() error {
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
