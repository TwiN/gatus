package email

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

type BatchConfig struct {
	Enabled bool          `yaml:"batch-enabled,omitempty"`
	Window  time.Duration `yaml:"batch-window,omitempty"`
}

func (cfg *BatchConfig) setDefaults() {
	if cfg.Window <= 0 {
		cfg.Window = 30 * time.Second
	}
}

type batchMessage struct {
	endpoint *endpoint.Endpoint
	alert    *alert.Alert
	result   *endpoint.Result
	resolved bool
}

type batchKey struct {
	from     string
	host     string
	port     int
	to       string
	resolved bool
}

type batchEntry struct {
	cfg      *Config
	messages []batchMessage
	timer    *time.Timer
}

var emailBatchState = struct {
	sync.Mutex
	entries map[batchKey]*batchEntry
}{entries: make(map[batchKey]*batchEntry)}

func (provider *AlertProvider) queueOrSend(ep *endpoint.Endpoint, al *alert.Alert, result *endpoint.Result, resolved bool, cfg *Config) error {
	if provider == nil || provider.Batch == nil || !provider.Batch.Enabled {
		return provider.sendEmail(cfg, provider.buildMessageSubjectAndBody(ep, al, result, resolved))
	}
	provider.Batch.setDefaults()
	message := batchMessage{endpoint: ep, alert: al, result: result, resolved: resolved}
	key := batchKey{from: cfg.From, host: cfg.Host, port: cfg.Port, to: cfg.To, resolved: resolved}

	emailBatchState.Lock()
	defer emailBatchState.Unlock()
	entry, exists := emailBatchState.entries[key]
	if !exists {
		entry = &batchEntry{cfg: cfg}
		entry.timer = time.AfterFunc(provider.Batch.Window, func() {
			provider.flushBatch(key)
		})
		emailBatchState.entries[key] = entry
	}
	entry.messages = append(entry.messages, message)
	return nil
}

func (provider *AlertProvider) flushBatch(key batchKey) {
	emailBatchState.Lock()
	entry, exists := emailBatchState.entries[key]
	if !exists {
		emailBatchState.Unlock()
		return
	}
	delete(emailBatchState.entries, key)
	emailBatchState.Unlock()

	subject, body := provider.buildBatchedMessageSubjectAndBody(entry.messages, key.resolved)
	_ = provider.sendEmail(entry.cfg, subject, body)
}

func (provider *AlertProvider) buildBatchedMessageSubjectAndBody(messages []batchMessage, resolved bool) (string, string) {
	if len(messages) == 1 {
		msg := messages[0]
		return provider.buildMessageSubjectAndBody(msg.endpoint, msg.alert, msg.result, resolved)
	}
	serviceNames := make([]string, 0, len(messages))
	for _, message := range messages {
		serviceNames = append(serviceNames, message.endpoint.DisplayName())
	}
	sort.Strings(serviceNames)
	statusWord := "triggered"
	intro := "The following alerts were triggered"
	if resolved {
		statusWord = "resolved"
		intro = "The following alerts were resolved"
	}
	subject := fmt.Sprintf("%d alerts %s", len(messages), statusWord)
	lines := []string{intro + ":", ""}
	for _, message := range messages {
		line := fmt.Sprintf("- %s", message.endpoint.DisplayName())
		if description := message.alert.GetDescription(); description != "" {
			line += fmt.Sprintf(" (%s)", description)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "", "Services: "+strings.Join(serviceNames, ", "))
	return subject, strings.Join(lines, "\n")
}
