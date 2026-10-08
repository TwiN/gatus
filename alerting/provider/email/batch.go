package email

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/logr"
)

const (
	DefaultBatchWindow = 2 * time.Minute
	DefaultBatchMax    = 50
)

type BatchConfig struct {
	Enabled bool          `yaml:"enabled,omitempty"`
	Window  time.Duration `yaml:"window,omitempty"`
	Max     int           `yaml:"max-alerts,omitempty"`
}

func (cfg *BatchConfig) setDefaults() {
	if cfg.Window <= 0 {
		cfg.Window = DefaultBatchWindow
	}
	if cfg.Max <= 0 {
		cfg.Max = DefaultBatchMax
	}
}

type batchMessage struct {
	endpoint *endpoint.Endpoint
	alert    *alert.Alert
	result   *endpoint.Result
	resolved bool
}

type batchKey struct {
	from string
	to   string
	host string
	port int
}

type batchManager struct {
	mu      sync.Mutex
	entries map[batchKey]*batchEntry
}

type batchEntry struct {
	cfg      *Config
	messages []batchMessage
	timer    *time.Timer
}

var emailBatchManager = &batchManager{entries: make(map[batchKey]*batchEntry)}

func (provider *AlertProvider) queueOrSend(ep *endpoint.Endpoint, al *alert.Alert, result *endpoint.Result, resolved bool, cfg *Config) error {
	if provider == nil || provider.Batch == nil || !provider.Batch.Enabled {
		subject, body := provider.buildMessageSubjectAndBody(ep, al, result, resolved)
		return provider.sendEmail(cfg, subject, body)
	}
	provider.Batch.setDefaults()
	message := batchMessage{endpoint: ep, alert: al, result: result, resolved: resolved}
	key := batchKey{from: cfg.From, to: cfg.To, host: cfg.Host, port: cfg.Port}
	return emailBatchManager.enqueue(provider, key, cfg, message)
}

func (manager *batchManager) enqueue(provider *AlertProvider, key batchKey, cfg *Config, message batchMessage) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	entry, exists := manager.entries[key]
	if !exists {
		entry = &batchEntry{cfg: cfg}
		entry.timer = time.AfterFunc(provider.Batch.Window, func() {
			manager.flush(provider, key)
		})
		manager.entries[key] = entry
	}
	entry.messages = appendOrReplace(entry.messages, message)
	if len(entry.messages) >= provider.Batch.Max {
		go manager.flush(provider, key)
	}
	return nil
}

func appendOrReplace(messages []batchMessage, message batchMessage) []batchMessage {
	for index, existing := range messages {
		if existing.endpoint.Key() == message.endpoint.Key() && existing.resolved == message.resolved {
			messages[index] = message
			return messages
		}
	}
	return append(messages, message)
}

func (manager *batchManager) flush(provider *AlertProvider, key batchKey) {
	entry := manager.take(key)
	if entry == nil || len(entry.messages) == 0 {
		return
	}
	subject, body := provider.buildBatchedMessageSubjectAndBody(entry.messages)
	if err := provider.sendEmail(entry.cfg, subject, body); err != nil {
		logr.Errorf("[email.batch.flush] Failed to send batched email alert: %s", err.Error())
	}
}

func (manager *batchManager) take(key batchKey) *batchEntry {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	entry, exists := manager.entries[key]
	if !exists {
		return nil
	}
	delete(manager.entries, key)
	if entry.timer != nil {
		entry.timer.Stop()
	}
	return entry
}

func (manager *batchManager) flushAll(provider *AlertProvider) {
	if provider == nil {
		return
	}
	manager.mu.Lock()
	keys := make([]batchKey, 0, len(manager.entries))
	for key := range manager.entries {
		keys = append(keys, key)
	}
	manager.mu.Unlock()
	for _, key := range keys {
		manager.flush(provider, key)
	}
}

func (provider *AlertProvider) buildBatchedMessageSubjectAndBody(messages []batchMessage) (string, string) {
	if len(messages) == 1 {
		msg := messages[0]
		return provider.buildMessageSubjectAndBody(msg.endpoint, msg.alert, msg.result, msg.resolved)
	}
	triggered := make([]batchMessage, 0)
	resolved := make([]batchMessage, 0)
	for _, message := range messages {
		if message.resolved {
			resolved = append(resolved, message)
		} else {
			triggered = append(triggered, message)
		}
	}
	subjectParts := make([]string, 0, 2)
	if len(triggered) > 0 {
		subjectParts = append(subjectParts, fmt.Sprintf("%d service(s) DOWN", len(triggered)))
	}
	if len(resolved) > 0 {
		subjectParts = append(subjectParts, fmt.Sprintf("%d service(s) recovered", len(resolved)))
	}
	if len(subjectParts) == 0 {
		subjectParts = append(subjectParts, fmt.Sprintf("%d alert(s)", len(messages)))
	}
	lines := []string{
		fmt.Sprintf("Gatus Alert Summary, %s", time.Now().UTC().Format("2006-01-02 15:04 UTC")),
		strings.Repeat("=", 60),
	}
	if len(triggered) > 0 {
		lines = append(lines, "", fmt.Sprintf("TRIGGERED (%d)", len(triggered)), strings.Repeat("-", 40))
		lines = append(lines, formatBatchLines(triggered, false)...)
	}
	if len(resolved) > 0 {
		lines = append(lines, "", fmt.Sprintf("RESOLVED (%d)", len(resolved)), strings.Repeat("-", 40))
		lines = append(lines, formatBatchLines(resolved, true)...)
	}
	lines = append(lines, "", "—", "Gatus")
	return fmt.Sprintf("[Gatus] %s", strings.Join(subjectParts, " / ")), strings.Join(lines, "\n")
}

func formatBatchLines(messages []batchMessage, resolved bool) []string {
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].endpoint.DisplayName() < messages[j].endpoint.DisplayName()
	})
	lines := make([]string, 0)
	for _, message := range messages {
		ep := message.endpoint
		lines = append(lines, fmt.Sprintf("- %s", ep.DisplayName()))
		if ep.Group != "" {
			lines = append(lines, fmt.Sprintf("  Group: %s", ep.Group))
		}
		if ep.URL != "" && !resolved {
			lines = append(lines, fmt.Sprintf("  URL: %s", ep.URL))
		}
		if description := message.alert.GetDescription(); description != "" {
			lines = append(lines, fmt.Sprintf("  Note: %s", description))
		}
		if !resolved && len(message.result.ConditionResults) > 0 {
			lines = append(lines, "  Conditions:")
			for _, conditionResult := range message.result.ConditionResults {
				prefix := "❌"
				if conditionResult.Success {
					prefix = "✅"
				}
				lines = append(lines, fmt.Sprintf("    %s %s", prefix, conditionResult.Condition))
			}
		}
		lines = append(lines, "")
	}
	return lines
}
