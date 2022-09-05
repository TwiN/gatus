package whois

import (
	"io"
	"net"
	"strings"
	"time"
)

const (
	ianaWHOISServerAddress = "whois.iana.org:43"
)

type Client struct {
	whoisServerAddress string
}

func NewClient() *Client {
	return &Client{
		whoisServerAddress: ianaWHOISServerAddress,
	}
}

func (c Client) Query(domain string) (string, error) {
	parts := strings.Split(domain, ".")
	output, err := c.query(c.whoisServerAddress, parts[len(parts)-1])
	if err != nil {
		return "", err
	}
	if strings.Contains(output, "whois:") {
		startIndex := strings.Index(output, "whois:") + 6
		endIndex := strings.Index(output[startIndex:], "\n") + startIndex
		whois := strings.TrimSpace(output[startIndex:endIndex])
		if referOutput, err := c.query(whois+":43", domain); err == nil {
			return referOutput, nil
		}
		return "", err
	}
	return output, nil
}

func (c Client) query(whoisServerAddress, domain string) (string, error) {
	connection, err := net.DialTimeout("tcp", whoisServerAddress, 10*time.Second)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	connection.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = connection.Write([]byte(domain + "\r\n"))
	if err != nil {
		return "", err
	}
	output, err := io.ReadAll(connection)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

type Response struct {
	ExpirationDate time.Time
	DomainStatuses []string
	NameServers    []string
}

// QueryAndParse tries to parse the response from the WHOIS server
// There is no standardized format for WHOIS responses, so this is an attempt at best.
//
// Being the selfish person that I am, I also only parse the fields that I need.
// If you need more fields, please open an issue or pull request.
func (c Client) QueryAndParse(domain string) (*Response, error) {
	text, err := c.Query(domain)
	if err != nil {
		return nil, err
	}
	response := Response{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		valueStartIndex := strings.Index(line, ":")
		if valueStartIndex == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:valueStartIndex]))
		value := strings.TrimSpace(line[valueStartIndex+1:])
		if response.ExpirationDate.Unix() != 0 && strings.Contains(key, "expir") && strings.Contains(key, "date") {
			response.ExpirationDate, _ = time.Parse(time.RFC3339, strings.ToUpper(value))
		} else if strings.Contains(key, "domain status") {
			response.DomainStatuses = append(response.DomainStatuses, value)
		} else if strings.Contains(key, "name server") {
			response.NameServers = append(response.NameServers, value)
		}
	}
	return &response, nil
}
