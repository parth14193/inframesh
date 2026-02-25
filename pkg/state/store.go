// Package state — file-backed store for session state and append-only audit log.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/parth14193/inframesh/pkg/core"
)

// FileStore persists SessionState and appends AuditEntry records to disk.
// It is safe for concurrent use.
type FileStore struct {
	mu          sync.Mutex
	stateFile   string // path to session.json
	auditFile   string // path to audit.log (JSON Lines)
	auditWriter *os.File
}

// NewFileStore creates a FileStore rooted in the given directory.
// The directory is created (with all parents) if it does not exist.
// Call Close() when done to flush the audit writer.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("state: cannot create directory %q: %w", dir, err)
	}

	auditPath := filepath.Join(dir, "audit.log")
	af, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, fmt.Errorf("state: cannot open audit log %q: %w", auditPath, err)
	}

	return &FileStore{
		stateFile:   filepath.Join(dir, "session.json"),
		auditFile:   auditPath,
		auditWriter: af,
	}, nil
}

// DefaultStoreDir returns ~/.infracore — the conventional home directory.
func DefaultStoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("state: cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".infracore"), nil
}

// SaveSession serialises the full SessionState to disk as JSON.
func (s *FileStore) SaveSession(state *core.SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal session: %w", err)
	}

	// Write to a temp file then rename to avoid partial writes.
	tmp := s.stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return fmt.Errorf("state: write session temp file: %w", err)
	}
	if err := os.Rename(tmp, s.stateFile); err != nil {
		return fmt.Errorf("state: rename session file: %w", err)
	}
	return nil
}

// LoadSession reads the session state from disk.
// Returns an empty SessionState (not an error) if no file exists yet.
func (s *FileStore) LoadSession() (*core.SessionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.stateFile)
	if os.IsNotExist(err) {
		return &core.SessionState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: read session file: %w", err)
	}

	var sess core.SessionState
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("state: unmarshal session: %w", err)
	}
	return &sess, nil
}

// AppendAudit writes a single AuditEntry as a JSON line to the audit log.
// The write is atomic at the OS level (single write syscall within the lock).
func (s *FileStore) AppendAudit(entry core.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("state: marshal audit entry: %w", err)
	}
	line = append(line, '\n')

	if _, err := s.auditWriter.Write(line); err != nil {
		return fmt.Errorf("state: write audit entry: %w", err)
	}
	return nil
}

// ReadAuditLog reads all persisted audit entries from the log file.
func (s *FileStore) ReadAuditLog() ([]core.AuditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.auditFile)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: read audit log: %w", err)
	}

	var entries []core.AuditEntry
	dec := json.NewDecoder(bytesReader(data))
	for dec.More() {
		var e core.AuditEntry
		if err := dec.Decode(&e); err != nil {
			return entries, fmt.Errorf("state: decode audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Close flushes and closes the audit log writer.
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.auditWriter != nil {
		return s.auditWriter.Close()
	}
	return nil
}

// AuditLogPath returns the path to the current audit log file.
func (s *FileStore) AuditLogPath() string { return s.auditFile }

// StateFilePath returns the path to the current session state file.
func (s *FileStore) StateFilePath() string { return s.stateFile }

// ─── helpers ───────────────────────────────────────────────────────────────

// bytesReader wraps a byte slice so json.NewDecoder can read it line-by-line.
type bytesReaderImpl struct {
	data   []byte
	offset int
}

func bytesReader(data []byte) *bytesReaderImpl { return &bytesReaderImpl{data: data} }

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

// ─── Manager integration ───────────────────────────────────────────────────

// PersistAuditEntry is a convenience wrapper that appends an AuditEntry to both
// the in-memory Manager and the FileStore in one call.  If store is nil the
// operation is a no-op for the file layer.
func PersistAuditEntry(
	mgr *Manager,
	store *FileStore,
	skillName, action, target string,
	status core.ExecutionStatus,
	riskLevel core.RiskLevel,
	details string,
) {
	mgr.AddToAuditLog(skillName, action, target, status, riskLevel, details)

	if store == nil {
		return
	}

	entry := core.AuditEntry{
		Timestamp: time.Now(),
		SkillName: skillName,
		Action:    action,
		Target:    target,
		Status:    status,
		RiskLevel: riskLevel,
		Details:   details,
	}
	// Best-effort — don't block the caller on I/O failures.
	_ = store.AppendAudit(entry)
}
