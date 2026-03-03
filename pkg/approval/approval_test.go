package approval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/parth14193/inframesh/pkg/core"
)

func TestCreateAndApprove(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	req, err := mgr.CreateRequest("k8s.deploy", "production", "dev-user", "deploy new version", core.RiskHigh, nil)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	if req.Status != StatusPending {
		t.Fatalf("expected PENDING, got %s", req.Status)
	}
	if req.Token == "" {
		t.Fatal("expected non-empty token")
	}

	approved, err := mgr.Approve(req.ID, "admin-user")
	if err != nil {
		t.Fatalf("failed to approve: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("expected APPROVED, got %s", approved.Status)
	}
	if approved.Approver != "admin-user" {
		t.Fatalf("expected approver admin-user, got %s", approved.Approver)
	}
}

func TestReject(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-reject-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	req, _ := mgr.CreateRequest("terraform.apply", "production", "user1", "infra change", core.RiskCritical, nil)

	rejected, err := mgr.Reject(req.ID, "too risky")
	if err != nil {
		t.Fatalf("failed to reject: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Fatalf("expected REJECTED, got %s", rejected.Status)
	}
	if rejected.RejectReason != "too risky" {
		t.Fatalf("expected reason 'too risky', got %s", rejected.RejectReason)
	}
}

func TestDoubleApproval(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-double-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	req, _ := mgr.CreateRequest("k8s.deploy", "staging", "user", "", core.RiskMedium, nil)
	_, _ = mgr.Approve(req.ID, "admin")
	_, err := mgr.Approve(req.ID, "admin2")
	if err == nil {
		t.Fatal("expected error on double approval")
	}
}

func TestListPending(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-list-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	mgr.CreateRequest("a", "prod", "u1", "", core.RiskHigh, nil)
	mgr.CreateRequest("b", "prod", "u2", "", core.RiskHigh, nil)
	req3, _ := mgr.CreateRequest("c", "prod", "u3", "", core.RiskHigh, nil)
	mgr.Approve(req3.ID, "admin")

	pending := mgr.ListPending()
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending, got %d", len(pending))
	}
}

func TestIsApproved(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-isapproved-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	req, _ := mgr.CreateRequest("k8s.deploy", "production", "dev", "", core.RiskHigh, nil)

	if mgr.IsApproved("k8s.deploy", "production") {
		t.Fatal("should not be approved yet")
	}

	mgr.Approve(req.ID, "admin")

	if !mgr.IsApproved("k8s.deploy", "production") {
		t.Fatal("should be approved now")
	}
}

func TestTokenVerification(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-token-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	req, _ := mgr.CreateRequest("k8s.deploy", "prod", "user", "", core.RiskHigh, nil)

	if !mgr.VerifyToken(req, req.Token) {
		t.Fatal("token should verify against the request")
	}

	if mgr.VerifyToken(req, "fake-token-abc") {
		t.Fatal("fake token should not verify")
	}
}

func TestPersistence(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-persist-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr1 := NewManagerAt(dir)
	mgr1.CreateRequest("k8s.deploy", "prod", "user", "", core.RiskHigh, nil)

	mgr2 := NewManagerAt(dir)
	all := mgr2.ListAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 persisted request, got %d", len(all))
	}
}

func TestCount(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-approval-count-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	mgr := NewManagerAt(dir)
	mgr.CreateRequest("a", "prod", "u1", "", core.RiskHigh, nil)
	req2, _ := mgr.CreateRequest("b", "prod", "u2", "", core.RiskHigh, nil)
	mgr.Approve(req2.ID, "admin")
	req3, _ := mgr.CreateRequest("c", "prod", "u3", "", core.RiskHigh, nil)
	mgr.Reject(req3.ID, "nope")

	p, a, r := mgr.Count()
	if p != 1 || a != 1 || r != 1 {
		t.Fatalf("expected 1/1/1, got %d/%d/%d", p, a, r)
	}
}
