// Package notify provides a notification system for sending alerts
// on skill execution outcomes to Slack, webhooks, and other channels.
package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// Event represents a notification event triggered by skill execution.
type Event struct {
	SkillName   string               `json:"skill_name"`
	Status      core.ExecutionStatus `json:"status"`
	Environment string               `json:"environment"`
	Provider    string               `json:"provider"`
	Region      string               `json:"region"`
	RiskLevel   core.RiskLevel       `json:"risk_level"`
	Message     string               `json:"message"`
	Duration    time.Duration        `json:"duration"`
	Timestamp   time.Time            `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Notifier defines the interface for sending notifications.
type Notifier interface {
	Send(event *Event) error
	Name() string
}

// ── Dispatcher ─────────────────────────────────────────────────

// Dispatcher routes events to multiple notification channels.
type Dispatcher struct {
	notifiers   []Notifier
	onSuccess   bool
	onFailure   bool
	onHighRisk  bool
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		notifiers:  []Notifier{},
		onSuccess:  true,
		onFailure:  true,
		onHighRisk: true,
	}
}

// AddNotifier registers a notification channel.
func (d *Dispatcher) AddNotifier(n Notifier) {
	d.notifiers = append(d.notifiers, n)
}

// SetFilters configures which events trigger notifications.
func (d *Dispatcher) SetFilters(onSuccess, onFailure, onHighRisk bool) {
	d.onSuccess = onSuccess
	d.onFailure = onFailure
	d.onHighRisk = onHighRisk
}

// Dispatch sends an event to all registered notifiers if it passes filters.
func (d *Dispatcher) Dispatch(event *Event) []error {
	if !d.shouldNotify(event) {
		return nil
	}

	var errs []error
	for _, n := range d.notifiers {
		if err := n.Send(event); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", n.Name(), err))
		}
	}
	return errs
}

func (d *Dispatcher) shouldNotify(event *Event) bool {
	if event.Status == core.StatusSuccess && d.onSuccess {
		return true
	}
	if event.Status == core.StatusFailed && d.onFailure {
		return true
	}
	if event.RiskLevel >= core.RiskHigh && d.onHighRisk {
		return true
	}
	return false
}

// ── Console Notifier ───────────────────────────────────────────

// ConsoleNotifier prints notifications to stdout.
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a new ConsoleNotifier.
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// Name returns "console".
func (c *ConsoleNotifier) Name() string { return "console" }

// Send prints the event to stdout.
func (c *ConsoleNotifier) Send(event *Event) error {
	icon := statusIcon(event.Status)
	fmt.Printf("\n🔔 NOTIFICATION %s\n", icon)
	fmt.Printf("   Skill:  %s\n", event.SkillName)
	fmt.Printf("   Status: %s\n", event.Status)
	fmt.Printf("   Env:    %s / %s / %s\n", event.Environment, event.Provider, event.Region)
	fmt.Printf("   Risk:   %s\n", event.RiskLevel)
	if event.Message != "" {
		fmt.Printf("   Msg:    %s\n", event.Message)
	}
	fmt.Printf("   Time:   %s (%s)\n", event.Timestamp.Format(time.RFC3339), event.Duration.Round(time.Millisecond))
	return nil
}

// ── Slack Notifier ─────────────────────────────────────────────

// SlackNotifier sends notifications to a Slack webhook.
type SlackNotifier struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// NewSlackNotifier creates a new SlackNotifier.
func NewSlackNotifier(webhookURL, channel string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "slack".
func (s *SlackNotifier) Name() string { return "slack" }

// Send posts the event to Slack via webhook.
func (s *SlackNotifier) Send(event *Event) error {
	icon := statusIcon(event.Status)
	riskEmoji := riskEmoji(event.RiskLevel)

	text := fmt.Sprintf("%s *InfraCore %s* — `%s`\n"+
		"• Environment: `%s / %s / %s`\n"+
		"• Risk Level: %s `%s`\n"+
		"• Duration: `%s`\n"+
		"• Message: %s",
		icon, event.Status, event.SkillName,
		event.Environment, event.Provider, event.Region,
		riskEmoji, event.RiskLevel,
		event.Duration.Round(time.Millisecond),
		event.Message,
	)

	payload := map[string]interface{}{
		"text": text,
	}
	if s.channel != "" {
		payload["channel"] = s.channel
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := s.httpClient.Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// ── Webhook Notifier ───────────────────────────────────────────

// WebhookNotifier sends notifications to a generic HTTP endpoint.
type WebhookNotifier struct {
	url        string
	headers    map[string]string
	httpClient *http.Client
}

// NewWebhookNotifier creates a new WebhookNotifier.
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier {
	return &WebhookNotifier{
		url:        url,
		headers:    headers,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Name returns "webhook".
func (w *WebhookNotifier) Name() string { return "webhook" }

// Send posts the event as JSON to the webhook endpoint.
func (w *WebhookNotifier) Send(event *Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// ── Helpers ────────────────────────────────────────────────────

func statusIcon(status core.ExecutionStatus) string {
	switch status {
	case core.StatusSuccess:
		return "✅"
	case core.StatusFailed:
		return "❌"
	case core.StatusDryRun:
		return "🧪"
	case core.StatusCancelled:
		return "🚫"
	default:
		return "📋"
	}
}

func riskEmoji(level core.RiskLevel) string {
	switch level {
	case core.RiskLow:
		return "🟢"
	case core.RiskMedium:
		return "🟡"
	case core.RiskHigh:
		return "🟠"
	case core.RiskCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

// FormatEventSummary creates a one-line summary of an event.
func FormatEventSummary(event *Event) string {
	icon := statusIcon(event.Status)
	return fmt.Sprintf("%s [%s] %s → %s (%s/%s) in %s",
		icon, event.RiskLevel, event.SkillName, event.Status, event.Environment,
		event.Region, event.Duration.Round(time.Millisecond))
}

// CreateEvent is a helper to build a notification event from an execution result.
func CreateEvent(result *core.ExecutionResult, env, provider, region string) *Event {
	return &Event{
		SkillName:   result.SkillName,
		Status:      result.Status,
		Environment: env,
		Provider:    provider,
		Region:      region,
		Message:     result.Message,
		Duration:    result.Duration,
		Timestamp:   result.Timestamp,
		Details:     result.Output,
	}
}

// Render formats all notifier names.
func (d *Dispatcher) Render() string {
	var b strings.Builder
	b.WriteString("🔔 NOTIFICATION CHANNELS\n")
	for _, n := range d.notifiers {
		b.WriteString(fmt.Sprintf("  • %s\n", n.Name()))
	}
	return b.String()
}
