package contract

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestOwnerAuditLogsScopedToOwner(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, ownerToken := newOwner(t, srv, pool)
	_, otherToken := newOwner(t, srv, pool)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPut, "/api/owner/bank-account", ownerToken,
		`{"bank_name":"BCA","account_number":"1234567890","account_name":"Pemilik Contoh"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put bank account: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/audit-logs?entity_type=owner_bank_account", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"action":"bank_account.update"`) ||
		!strings.Contains(string(body), fmt.Sprintf(`"actor":{"id":%d`, ownerID)) {
		t.Errorf("owner audit logs = %s, want bank_account.update by owner %d", body, ownerID)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/audit-logs?entity_type=owner_bank_account", otherToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("other owner list: status = %d, body = %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "bank_account.update") {
		t.Errorf("other owner sees entry it does not own: %s", body)
	}
}

func TestAdminAuditLogsIncludeIPAndUserAgent(t *testing.T) {
	srv, _ := newTestServerAndPool(t)
	token := adminToken(t, srv)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPatch, "/api/admin/settings", token, `{"admin_fee":7000}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch settings: status = %d, body = %s", resp.StatusCode, body)
	}
	t.Cleanup(func() {
		bookingAuthedRequest(t, srv, http.MethodPatch, "/api/admin/settings", token, `{"admin_fee":0}`)
	})

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/audit-logs?action=settings.update&entity_type=setting", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}
	var list struct {
		AuditLogs []struct {
			Action     string `json:"action"`
			EntityType string `json:"entity_type"`
			UserAgent  string `json:"user_agent"`
		} `json:"audit_logs"`
	}
	mustUnmarshal(t, body, &list)
	var found bool
	for _, entry := range list.AuditLogs {
		if entry.Action == "settings.update" {
			found = true
			if entry.UserAgent == "" {
				t.Errorf("entry = %+v, want non-empty user_agent", entry)
			}
		}
	}
	if !found {
		t.Errorf("admin audit logs %s missing settings.update", body)
	}
	if !strings.Contains(string(body), `"ip":`) {
		t.Errorf("admin audit log entries missing ip field: %s", body)
	}
}
