package endpoint

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	sshconfig "github.com/TwiN/gatus/v5/config/endpoint/ssh"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/config/key"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"golang.org/x/crypto/ssh"
)

type Type string

const (
	// HostHeader is the name of the header used to specify the host
	HostHeader = "Host"

	// ContentTypeHeader is the name of the header used to specify the content type
	ContentTypeHeader = "Content-Type"

	// UserAgentHeader is the name of the header used to specify the request's user agent
	UserAgentHeader = "User-Agent"

	// GatusUserAgent is the default user agent that Gatus uses to send requests.
	GatusUserAgent = "Gatus/1.0"

	TypeDNS      Type = "DNS"
	TypeTCP      Type = "TCP"
	TypeSCTP     Type = "SCTP"
	TypeUDP      Type = "UDP"
	TypeICMP     Type = "ICMP"
	TypeSTARTTLS Type = "STARTTLS"
	TypeTLS      Type = "TLS"
	TypeHTTP     Type = "HTTP"
	TypeGRPC     Type = "GRPC"
	TypeWS       Type = "WEBSOCKET"
	TypeSSH      Type = "SSH"
	TypeUNKNOWN  Type = "UNKNOWN"
)

var (
	// ErrEndpointWithNoCondition is the error with which Gatus will panic if an endpoint is configured with no conditions
	ErrEndpointWithNoCondition = errors.New("you must specify at least one condition per endpoint")

	// ErrEndpointWithNoURL is the error with which Gatus will panic if an endpoint is configured with no url
	ErrEndpointWithNoURL = errors.New("you must specify an url for each endpoint")

	// ErrUnknownEndpointType is the error with which Gatus will panic if an endpoint has an unknown type
	ErrUnknownEndpointType = errors.New("unknown endpoint type")

	// ErrInvalidConditionFormat is the error with which Gatus will panic if a condition has an invalid format
	ErrInvalidConditionFormat = errors.New("invalid condition format: does not match '<VALUE> <COMPARATOR> <VALUE>'")

	// ErrInvalidEndpointIntervalForDomainExpirationPlaceholder is the error with which Gatus will panic if an endpoint
	// has both an interval smaller than 5 minutes and a condition with DomainExpirationPlaceholder.
	// This is because the free whois service we are using should not be abused, especially considering the fact that
	// the data takes a while to be updated.
	ErrInvalidEndpointIntervalForDomainExpirationPlaceholder = errors.New("the minimum interval for an endpoint with a condition using the " + DomainExpirationPlaceholder + " placeholder is 300s (5m)")
)

// Endpoint is the configuration of a service to be monitored
type Endpoint struct {
	// Enabled defines whether to enable the monitoring of the endpoint
	Enabled *bool `yaml:"enabled,omitempty"`

	// Name of the endpoint. Can be anything.
	Name string `yaml:"name"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `yaml:"group,omitempty"`

	// URL to send the request to
	URL string `yaml:"url"`

	// Method of the request made to the url of the endpoint
	Method string `yaml:"method,omitempty"`

	// Body of the request
	Body string `yaml:"body,omitempty"`

	// GraphQL is whether to wrap the body in a query param ({"query":"$body"})
	GraphQL bool `yaml:"graphql,omitempty"`

	// Headers of the request
	Headers map[string]string `yaml:"headers,omitempty"`

	// ExtraLabels are key-value pairs that can be used to metric the endpoint
	ExtraLabels map[string]string `yaml:"extra-labels,omitempty"`

	// Interval is the duration to wait between every status check
	Interval time.Duration `yaml:"interval,omitempty"`

	// Conditions used to determine the health of the endpoint
	Conditions []Condition `yaml:"conditions"`

	// Alerts is the alerting configuration for the endpoint in case of failure
	Alerts []*alert.Alert `yaml:"alerts,omitempty"`

	// MaintenanceWindow is the configuration for per-endpoint maintenance windows
	MaintenanceWindows []*maintenance.Config `yaml:"maintenance-windows,omitempty"`

	// DNSConfig is the configuration for DNS monitoring
	DNSConfig *dns.Config `yaml:"dns,omitempty"`

	// SSH is the configuration for SSH monitoring
	SSHConfig *sshconfig.Config `yaml:"ssh,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the endpoint's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// UIConfig is the configuration for the UI
	UIConfig *ui.Config `yaml:"ui,omitempty"`

	// NumberOfFailuresInARow is the number of unsuccessful evaluations in a row
	NumberOfFailuresInARow int `yaml:"-"`

	// NumberOfSuccessesInARow is the number of successful evaluations in a row
	NumberOfSuccessesInARow int `yaml:"-"`

	// LastReminderSent is the time at which the last reminder was sent for this endpoint.
	LastReminderSent time.Time `yaml:"-"`

	///////////////////////
	// SUITE-ONLY FIELDS //
	///////////////////////

	// Store is a map of values to extract from the result and store in the suite context
	// This field is only used when the endpoint is part of a suite
	Store map[string]string `yaml:"store,omitempty"`

	// AlwaysRun defines whether to execute this endpoint even if previous endpoints in the suite failed
	// This field is only used when the endpoint is part of a suite
	AlwaysRun bool `yaml:"always-run,omitempty"`
}

// IsEnabled returns whether the endpoint is enabled or not
func (e *Endpoint) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

// Type returns the endpoint type
func (e *Endpoint) Type() Type {
	switch {
	case e.DNSConfig != nil:
		return TypeDNS
	case strings.HasPrefix(e.URL, "tcp://"):
		return TypeTCP
	case strings.HasPrefix(e.URL, "sctp://"):
		return TypeSCTP
	case strings.HasPrefix(e.URL, "udp://"):
		return TypeUDP
	case strings.HasPrefix(e.URL, "icmp://"):
		return TypeICMP
	case strings.HasPrefix(e.URL, "starttls://"):
		return TypeSTARTTLS
	case strings.HasPrefix(e.URL, "tls://"):
		return TypeTLS
	case strings.HasPrefix(e.URL, "http://") || strings.HasPrefix(e.URL, "https://"):
		return TypeHTTP
	case strings.HasPrefix(e.URL, "grpc://") || strings.HasPrefix(e.URL, "grpcs://"):
		return TypeGRPC
	case strings.HasPrefix(e.URL, "ws://") || strings.HasPrefix(e.URL, "wss://"):
		return TypeWS
	case strings.HasPrefix(e.URL, "ssh://"):
		return TypeSSH
	default:
		return TypeUNKNOWN
	}
}

// ValidateAndSetDefaults validates the endpoint's configuration and sets the default value of args that have one
func (e *Endpoint) ValidateAndSetDefaults() error {
	if err := validateEndpointNameGroupAndAlerts(e.Name, e.Group, e.Alerts); err != nil {
		return err
	}
	if len(e.URL) == 0 {
		return ErrEndpointWithNoURL
	}
	if e.ClientConfig == nil {
		e.ClientConfig = client.GetDefaultConfig()
	} else {
		if err := e.ClientConfig.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if e.UIConfig == nil {
		e.UIConfig = ui.GetDefaultConfig()
	} else {
		if err := e.UIConfig.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if e.Interval == 0 {
		e.Interval = 1 * time.Minute
	}
	if len(e.Method) == 0 {
		e.Method = http.MethodGet
	}
	if len(e.Headers) == 0 {
		e.Headers = make(map[string]string)
	}
	// Automatically add user agent header if there isn't one specified in the endpoint configuration
	if _, userAgentHeaderExists := e.Headers[UserAgentHeader]; !userAgentHeaderExists {
		e.Headers[UserAgentHeader] = GatusUserAgent
	}
	// Automatically add "Content-Type: application/json" header if there's no Content-Type set
	// and endpoint.GraphQL is set to true
	if _, contentTypeHeaderExists := e.Headers[ContentTypeHeader]; !contentTypeHeaderExists && e.GraphQL {
		e.Headers[ContentTypeHeader] = "application/json"
	}
	if len(e.Conditions) == 0 {
		return ErrEndpointWithNoCondition
	}
	for _, c := range e.Conditions {
		if e.Interval < 5*time.Minute && c.hasDomainExpirationPlaceholder() {
			return ErrInvalidEndpointIntervalForDomainExpirationPlaceholder
		}
		if err := c.Validate(); err != nil {
			return fmt.Errorf("%v: %w", ErrInvalidConditionFormat, err)
		}
	}
	if e.DNSConfig != nil {
		return e.DNSConfig.ValidateAndSetDefault()
	}
	if e.SSHConfig != nil {
		return e.SSHConfig.Validate()
	}
	if e.Type() == TypeUNKNOWN {
		return ErrUnknownEndpointType
	}
	for _, maintenanceWindow := range e.MaintenanceWindows {
		if err := maintenanceWindow.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	// Make sure that the request can be created
	_, err := http.NewRequest(e.Method, e.URL, bytes.NewBuffer([]byte(e.getParsedBody())))
	if err != nil {
		return err
	}
	return nil
}

// DisplayName returns an identifier made up of the Name and, if not empty, the Group.
func (e *Endpoint) DisplayName() string {
	if len(e.Group) > 0 {
		return e.Group + "/" + e.Name
	}
	return e.Name
}

// Key returns the unique key for the Endpoint
func (e *Endpoint) Key() string {
	return key.ConvertGroupAndNameToKey(e.Group, e.Name)
}

// Close HTTP connections between watchdog and endpoints to avoid dangling socket file descriptors
// on configuration reload.
// More context on https://github.com/TwiN/gatus/issues/536
func (e *Endpoint) Close() {
	if e.Type() == TypeHTTP {
		client.GetHTTPClient(e.ClientConfig).CloseIdleConnections()
	}
}

// EvaluateHealth sends a request to the endpoint's URL and evaluates the conditions of the endpoint.
func (e *Endpoint) EvaluateHealth() *Result {
	return e.EvaluateHealthWithContext(nil)
}

// EvaluateHealthWithContext sends a request to the endpoint's URL with context support and evaluates the conditions
func (e *Endpoint) EvaluateHealthWithContext(context *gontext.Gontext) *Result {
	result := &Result{Success: true, Errors: []string{}}
	// Preprocess the endpoint with context if provided
	processedEndpoint := e
	if context != nil {
		processedEndpoint = e.preprocessWithContext(result, context)
	}
	// Parse or extract hostname from URL
	if processedEndpoint.DNSConfig != nil {
		result.Hostname = strings.TrimSuffix(processedEndpoint.URL, ":53")
	} else if processedEndpoint.Type() == TypeICMP {
		// To handle IPv6 addresses, we need to handle the hostname differently here. This is to avoid, for instance,
		// "1111:2222:3333::4444" being displayed as "1111:2222:3333:" because :4444 would be interpreted as a port.
		result.Hostname = strings.TrimPrefix(processedEndpoint.URL, "icmp://")
	} else {
		urlObject, err := url.Parse(processedEndpoint.URL)
		if err != nil {
			result.AddError(err.Error())
		} else {
			result.Hostname = urlObject.Hostname()
			result.port = urlObject.Port()
		}
	}
	// Retrieve IP if necessary
	if processedEndpoint.needsToRetrieveIP() {
		processedEndpoint.getIP(result)
	}
	// Retrieve domain expiration if necessary
	if processedEndpoint.needsToRetrieveDomainExpiration() && len(result.Hostname) > 0 {
		var err error
		if result.DomainExpiration, err = client.GetDomainExpiration(result.Hostname); err != nil {
			result.AddError(err.Error())
		}
	}
	// Call the endpoint (if there's no errors)
	if len(result.Errors) == 0 {
		processedEndpoint.call(result)
	} else {
		result.Success = false
	}
	// Evaluate the conditions
	for _, condition := range processedEndpoint.Conditions {
		success := condition.evaluate(result, processedEndpoint.UIConfig.DontResolveFailedConditions, processedEndpoint.UIConfig.ResolveSuccessfulConditions, context)
		if !success {
			result.Success = false
		}
	}
	result.Timestamp = time.Now()
	// Clean up parameters that we don't need to keep in the results
	if processedEndpoint.UIConfig.HideURL {
		for errIdx, errorString := range result.Errors {
			result.Errors[errIdx] = strings.ReplaceAll(errorString, processedEndpoint.URL, "<redacted>")
		}
	}
	if processedEndpoint.UIConfig.HideHostname {
		for errIdx, errorString := range result.Errors {
			result.Errors[errIdx] = strings.ReplaceAll(errorString, result.Hostname, "<redacted>")
		}
		result.Hostname = "" // remove it from the result so it doesn't get exposed
	}
	if processedEndpoint.UIConfig.HidePort && len(result.port) > 0 {
		for errIdx, errorString := range result.Errors {
			result.Errors[errIdx] = strings.ReplaceAll(errorString, result.port, "<redacted>")
		}
		result.port = ""
	}
	if processedEndpoint.UIConfig.HideErrors {
		result.Errors = nil
	}
	if processedEndpoint.UIConfig.HideConditions {
		result.ConditionResults = nil
	}
	return result
}

// preprocessWithContext creates a copy of the endpoint with context placeholders replaced
func (e *Endpoint) preprocessWithContext(result *Result, context *gontext.Gontext) *Endpoint {
	// Create a deep copy of the endpoint
	processed := &Endpoint{}
	*processed = *e
	var err error
	// Replace context placeholders in URL
	if processed.URL, err = replaceContextPlaceholders(e.URL, context); err != nil {
		result.AddError(err.Error())
	}
	// Replace context placeholders in Body
	if processed.Body, err = replaceContextPlaceholders(e.Body, context); err != nil {
		result.AddError(err.Error())
	}
	// Replace context placeholders in Headers
	if e.Headers != nil {
		processed.Headers = make(map[string]string)
		for k, v := range e.Headers {
			if processed.Headers[k], err = replaceContextPlaceholders(v, context); err != nil {
				result.AddError(err.Error())
			}
		}
	}
	return processed
}

// replaceContextPlaceholders replaces [CONTEXT].path placeholders with actual values
func replaceContextPlaceholders(input string, ctx *gontext.Gontext) (string, error) {
	if ctx == nil {
		return input, nil
	}
	var contextErrors []string
	contextRegex := regexp.MustCompile(`\[CONTEXT\]\.[\w\.\-]+`)
	result := contextRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract the path after [CONTEXT].
		path := strings.TrimPrefix(match, "[CONTEXT].")
		value, err := ctx.Get(path)
		if err != nil {
			contextErrors = append(contextErrors, fmt.Sprintf("path '%s' not found", path))
			return match // Keep placeholder for error reporting
		}
		return fmt.Sprintf("%v", value)
	})
	if len(contextErrors) > 0 {
		return result, fmt.Errorf("context placeholder resolution failed: %s", strings.Join(contextErrors, ", "))
	}
	return result, nil
}

func (e *Endpoint) getParsedBody() string {
	body := e.Body
	body = strings.ReplaceAll(body, "[ENDPOINT_NAME]", e.Name)
	body = strings.ReplaceAll(body, "[ENDPOINT_GROUP]", e.Group)
	body = strings.ReplaceAll(body, "[ENDPOINT_URL]", e.URL)
	randRegex, err := regexp.Compile(`\[RANDOM_STRING_\d+\]`)
	if err == nil {
		body = randRegex.ReplaceAllStringFunc(body, func(match string) string {
			n, _ := strconv.Atoi(match[15 : len(match)-1])
			if n > 8192 {
				n = 8192 // Limit the length of the random string to 8192 bytes to avoid excessive memory usage
			}
			const availableCharacterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			b := make([]byte, n)
			for i := range b {
				b[i] = availableCharacterBytes[rand.Intn(len(availableCharacterBytes))]
			}
			return string(b)
		})
	}
	return body
}

func (e *Endpoint) getIP(result *Result) {
	if ips, err := net.LookupIP(result.Hostname); err != nil {
		result.AddError(err.Error())
		return
	} else {
		result.IP = ips[0].String()
	}
}

func (e *Endpoint) call(result *Result) {
	var request *http.Request
	var response *http.Response
	var err error
	var certificate *x509.Certificate
	endpointType := e.Type()
	if endpointType == TypeHTTP {
		request = e.buildHTTPRequest()
	}
	startTime := time.Now()
	if endpointType == TypeDNS {
		result.Connected, result.DNSRCode, result.Body, err = client.QueryDNS(e.DNSConfig.QueryType, e.DNSConfig.QueryName, e.URL)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeSTARTTLS || endpointType == TypeTLS {
		if endpointType == TypeSTARTTLS {
			result.Connected, certificate, err = client.CanPerformStartTLS(strings.TrimPrefix(e.URL, "starttls://"), e.ClientConfig)
		} else {
			result.Connected, result.Body, certificate, err = client.CanPerformTLS(strings.TrimPrefix(e.URL, "tls://"), e.getParsedBody(), e.ClientConfig)
		}
		if err != nil {
			result.AddError(err.Error())
			return
		}
		result.Duration = time.Since(startTime)
		result.CertificateExpiration = time.Until(certificate.NotAfter)
	} else if endpointType == TypeTCP {
		result.Connected, result.Body = client.CanCreateNetworkConnection("tcp", strings.TrimPrefix(e.URL, "tcp://"), e.getParsedBody(), e.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeUDP {
		result.Connected, result.Body = client.CanCreateNetworkConnection("udp", strings.TrimPrefix(e.URL, "udp://"), e.getParsedBody(), e.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeSCTP {
		result.Connected = client.CanCreateSCTPConnection(strings.TrimPrefix(e.URL, "sctp://"), e.ClientConfig)
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeICMP {
		result.Connected, result.Duration = client.Ping(strings.TrimPrefix(e.URL, "icmp://"), e.ClientConfig)
	} else if endpointType == TypeWS {
		wsHeaders := map[string]string{}
		if e.Headers != nil {
			for k, v := range e.Headers {
				wsHeaders[k] = v
			}
		}
		if _, exists := wsHeaders["User-Agent"]; !exists {
			wsHeaders["User-Agent"] = GatusUserAgent
		}
		result.Connected, result.Body, err = client.QueryWebSocket(e.URL, e.getParsedBody(), wsHeaders, e.ClientConfig)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeSSH {
		// If there's no username, password or private key specified, attempt to validate just the SSH banner
		if e.SSHConfig == nil || (len(e.SSHConfig.Username) == 0 && len(e.SSHConfig.Password) == 0 && len(e.SSHConfig.PrivateKey) == 0) {
			result.Connected, result.HTTPStatus, err = client.CheckSSHBanner(strings.TrimPrefix(e.URL, "ssh://"), e.ClientConfig)
			if err != nil {
				result.AddError(err.Error())
				return
			}
			result.Success = result.Connected
			result.Duration = time.Since(startTime)
			return
		}
		var cli *ssh.Client
		result.Connected, cli, err = client.CanCreateSSHConnection(strings.TrimPrefix(e.URL, "ssh://"), e.SSHConfig.Username, e.SSHConfig.Password, e.SSHConfig.PrivateKey, e.ClientConfig)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		var output []byte
		result.Success, result.HTTPStatus, output, err = client.ExecuteSSHCommand(cli, e.getParsedBody(), e.ClientConfig)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		// Only store the output in result.Body if there's a condition that uses the BodyPlaceholder
		if e.needsToReadBody() {
			result.Body = output
		}
		result.Duration = time.Since(startTime)
	} else if endpointType == TypeGRPC {
		useTLS := strings.HasPrefix(e.URL, "grpcs://")
		address := strings.TrimPrefix(strings.TrimPrefix(e.URL, "grpcs://"), "grpc://")
		connected, status, err, duration := client.PerformGRPCHealthCheck(address, useTLS, e.ClientConfig)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		result.Connected = connected
		result.Duration = duration
		if e.needsToReadBody() {
			result.Body = []byte(fmt.Sprintf("{\"status\":\"%s\"}", status))
		}
	} else {
		response, err = client.GetHTTPClient(e.ClientConfig).Do(request)
		result.Duration = time.Since(startTime)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		defer response.Body.Close()
		if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
			certificate = response.TLS.PeerCertificates[0]
			result.CertificateExpiration = time.Until(certificate.NotAfter)
		}
		result.HTTPStatus = response.StatusCode
		result.Connected = response.StatusCode > 0
		// Only read the Body if there's a condition that uses the BodyPlaceholder
		if e.needsToReadBody() {
			result.Body, err = io.ReadAll(response.Body)
			if err != nil {
				result.AddError("error reading response body:" + err.Error())
			}
		}
	}
}

func (e *Endpoint) buildHTTPRequest() *http.Request {
	var bodyBuffer *bytes.Buffer
	if e.GraphQL {
		graphQlBody := map[string]string{
			"query": e.getParsedBody(),
		}
		body, _ := json.Marshal(graphQlBody)
		bodyBuffer = bytes.NewBuffer(body)
	} else {
		bodyBuffer = bytes.NewBuffer([]byte(e.getParsedBody()))
	}
	request, _ := http.NewRequest(e.Method, e.URL, bodyBuffer)
	for k, v := range e.Headers {
		request.Header.Set(k, v)
		if k == HostHeader {
			request.Host = v
		}
	}
	return request
}

// needsToReadBody checks if there's any condition or store mapping that requires the response Body to be read
func (e *Endpoint) needsToReadBody() bool {
	for _, condition := range e.Conditions {
		if condition.hasBodyPlaceholder() {
			return true
		}
	}
	// Check store values for body placeholders
	if e.Store != nil {
		for _, value := range e.Store {
			if strings.Contains(value, BodyPlaceholder) {
				return true
			}
		}
	}
	return false
}

// needsToRetrieveDomainExpiration checks if there's any condition that requires a whois query to be performed
func (e *Endpoint) needsToRetrieveDomainExpiration() bool {
	for _, condition := range e.Conditions {
		if condition.hasDomainExpirationPlaceholder() {
			return true
		}
	}
	return false
}

// needsToRetrieveIP checks if there's any condition that requires an IP lookup
func (e *Endpoint) needsToRetrieveIP() bool {
	for _, condition := range e.Conditions {
		if condition.hasIPPlaceholder() {
			return true
		}
	}
	return false
}
