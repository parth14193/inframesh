// Package health provides infrastructure health monitoring with
// configurable probes for HTTP, TCP, Kubernetes, and DNS endpoints.
package health

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ProbeType defines what kind of health check to perform.
type ProbeType string

const (
	ProbeHTTP ProbeType = "http"
	ProbeTCP  ProbeType = "tcp"
	ProbeK8s  ProbeType = "k8s"
	ProbeDNS  ProbeType = "dns"
)

// ProbeStatus represents the status of a health check.
type ProbeStatus string

const (
	StatusHealthy   ProbeStatus = "HEALTHY"
	StatusDegraded  ProbeStatus = "DEGRADED"
	StatusUnhealthy ProbeStatus = "UNHEALTHY"
	StatusUnknown   ProbeStatus = "UNKNOWN"
)

// Probe defines a single health check configuration.
type Probe struct {
	Name           string        `json:"name"`
	Type           ProbeType     `json:"type"`
	Target         string        `json:"target"`
	Interval       time.Duration `json:"interval"`
	Timeout        time.Duration `json:"timeout"`
	ExpectedStatus int           `json:"expected_status,omitempty"`
	Tags           []string      `json:"tags,omitempty"`
}

// ProbeResult is the outcome of a single probe execution.
type ProbeResult struct {
	ProbeName  string        `json:"probe_name"`
	Status     ProbeStatus   `json:"status"`
	Latency    time.Duration `json:"latency"`
	StatusCode int           `json:"status_code,omitempty"`
	Message    string        `json:"message"`
	Timestamp  time.Time     `json:"timestamp"`
	Error      string        `json:"error,omitempty"`
}

// HealthReport aggregates all probe results.
type HealthReport struct {
	Timestamp time.Time     `json:"timestamp"`
	Overall   ProbeStatus   `json:"overall"`
	Results   []ProbeResult `json:"results"`
	Healthy   int           `json:"healthy"`
	Degraded  int           `json:"degraded"`
	Unhealthy int           `json:"unhealthy"`
}

// Checker runs health probes and aggregates results.
type Checker struct {
	probes     []*Probe
	httpClient *http.Client
}

// NewChecker creates a new HealthChecker.
func NewChecker() *Checker {
	return &Checker{
		probes:     []*Probe{},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// AddProbe registers a health probe.
func (c *Checker) AddProbe(probe *Probe) {
	if probe.Timeout == 0 {
		probe.Timeout = 5 * time.Second
	}
	if probe.Interval == 0 {
		probe.Interval = 30 * time.Second
	}
	c.probes = append(c.probes, probe)
}

// RunAll executes all probes and returns an aggregate report.
func (c *Checker) RunAll() *HealthReport {
	report := &HealthReport{Timestamp: time.Now()}
	for _, probe := range c.probes {
		result := c.runProbe(probe)
		report.Results = append(report.Results, result)
		switch result.Status {
		case StatusHealthy:
			report.Healthy++
		case StatusDegraded:
			report.Degraded++
		case StatusUnhealthy:
			report.Unhealthy++
		}
	}
	report.Overall = StatusHealthy
	if report.Degraded > 0 {
		report.Overall = StatusDegraded
	}
	if report.Unhealthy > 0 {
		report.Overall = StatusUnhealthy
	}
	return report
}

// RunByTag executes probes matching a tag.
func (c *Checker) RunByTag(tag string) *HealthReport {
	report := &HealthReport{Timestamp: time.Now()}
	for _, probe := range c.probes {
		if !hasTag(probe.Tags, tag) {
			continue
		}
		result := c.runProbe(probe)
		report.Results = append(report.Results, result)
		switch result.Status {
		case StatusHealthy:
			report.Healthy++
		case StatusDegraded:
			report.Degraded++
		case StatusUnhealthy:
			report.Unhealthy++
		}
	}
	report.Overall = StatusHealthy
	if report.Degraded > 0 {
		report.Overall = StatusDegraded
	}
	if report.Unhealthy > 0 {
		report.Overall = StatusUnhealthy
	}
	return report
}

func (c *Checker) runProbe(probe *Probe) ProbeResult {
	switch probe.Type {
	case ProbeHTTP:
		return c.runHTTP(probe)
	case ProbeTCP:
		return c.runTCP(probe)
	case ProbeDNS:
		return c.runDNS(probe)
	default:
		return ProbeResult{ProbeName: probe.Name, Status: StatusUnknown, Message: "Unknown probe type", Timestamp: time.Now()}
	}
}

func (c *Checker) runHTTP(probe *Probe) ProbeResult {
	start := time.Now()
	client := &http.Client{Timeout: probe.Timeout}
	resp, err := client.Get(probe.Target)
	result := ProbeResult{ProbeName: probe.Name, Latency: time.Since(start), Timestamp: time.Now()}
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		result.Message = "HTTP request failed"
		return result
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	expected := probe.ExpectedStatus
	if expected == 0 {
		expected = 200
	}
	if resp.StatusCode == expected {
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("HTTP %d (%s)", resp.StatusCode, result.Latency.Round(time.Millisecond))
	} else if resp.StatusCode >= 500 {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, expected)
	}
	if result.Status == StatusHealthy && result.Latency > 2*time.Second {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("High latency: %s", result.Latency.Round(time.Millisecond))
	}
	return result
}

func (c *Checker) runTCP(probe *Probe) ProbeResult {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", probe.Target, probe.Timeout)
	result := ProbeResult{ProbeName: probe.Name, Latency: time.Since(start), Timestamp: time.Now()}
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		result.Message = "TCP connect failed"
		return result
	}
	conn.Close()
	result.Status = StatusHealthy
	result.Message = fmt.Sprintf("TCP connected (%s)", result.Latency.Round(time.Millisecond))
	return result
}

func (c *Checker) runDNS(probe *Probe) ProbeResult {
	start := time.Now()
	ips, err := net.LookupHost(probe.Target)
	result := ProbeResult{ProbeName: probe.Name, Latency: time.Since(start), Timestamp: time.Now()}
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
		result.Message = "DNS lookup failed"
		return result
	}
	result.Status = StatusHealthy
	result.Message = fmt.Sprintf("Resolved %d IPs (%s)", len(ips), result.Latency.Round(time.Millisecond))
	return result
}

// ListProbes returns all probes.
func (c *Checker) ListProbes() []*Probe { return c.probes }

// LoadBuiltins registers default probes.
func (c *Checker) LoadBuiltins() {
	c.AddProbe(&Probe{Name: "dns-google", Type: ProbeDNS, Target: "google.com", Tags: []string{"dns"}})
	c.AddProbe(&Probe{Name: "dns-aws", Type: ProbeDNS, Target: "aws.amazon.com", Tags: []string{"dns", "cloud"}})
}

func hasTag(tags []string, target string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}

// Render formats a health report for display.
func (r *HealthReport) Render() string {
	var b strings.Builder
	icon := statusIcon(r.Overall)
	b.WriteString(fmt.Sprintf("🏥 HEALTH CHECK %s\n", icon))
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Overall: %s %s\n", icon, r.Overall))
	b.WriteString(fmt.Sprintf("  ✅ Healthy: %d  ⚠️ Degraded: %d  ❌ Unhealthy: %d\n\n", r.Healthy, r.Degraded, r.Unhealthy))
	for _, pr := range r.Results {
		i := statusIcon(pr.Status)
		b.WriteString(fmt.Sprintf("  %s %-20s %s (%s)\n", i, pr.ProbeName, pr.Status, pr.Latency.Round(time.Millisecond)))
		if pr.Error != "" {
			b.WriteString(fmt.Sprintf("     ❗ %s\n", pr.Error))
		}
	}
	return b.String()
}

func statusIcon(s ProbeStatus) string {
	switch s {
	case StatusHealthy:
		return "✅"
	case StatusDegraded:
		return "⚠️"
	case StatusUnhealthy:
		return "❌"
	default:
		return "❓"
	}
}
