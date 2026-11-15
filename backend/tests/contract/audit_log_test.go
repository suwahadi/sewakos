package contract

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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

// API-ADMIN.md §12: the filter options list every actor and action present in the audit log, and both narrow the list.
func TestAdminAuditLogFilters(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	phone, _ := bookingRegisterAndVerify(t, srv)
	var tenantID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE phone = $1`, phone).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/admin/users/%d/password", tenantID), token,
		`{"password":"Baru12345","password_confirmation":"Baru12345"}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("set password: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/audit-logs/filters", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filters: status = %d, body = %s", resp.StatusCode, body)
	}
	var filters struct {
		Actors []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		} `json:"actors"`
		Actions []string `json:"actions"`
	}
	mustUnmarshal(t, body, &filters)
	var adminID int64
	for _, actor := range filters.Actors {
		if actor.Role == "admin" && actor.Name != "" {
			adminID = actor.ID
		}
	}
	if adminID == 0 {
		t.Fatalf("actors %+v miss the admin who set the password", filters.Actors)
	}
	if !slices.Contains(filters.Actions, "user.password_set") {
		t.Fatalf("actions %v miss user.password_set", filters.Actions)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet,
		fmt.Sprintf("/api/admin/audit-logs?actor_id=%d&action=user.password_set&limit=50", adminID), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("filtered list: status = %d, body = %s", resp.StatusCode, body)
	}
	var list struct {
		Logs []struct {
			Action string `json:"action"`
			Actor  struct {
				ID int64 `json:"id"`
			} `json:"actor"`
		} `json:"audit_logs"`
	}
	mustUnmarshal(t, body, &list)
	if len(list.Logs) == 0 {
		t.Fatal("filtered list is empty")
	}
	for _, entry := range list.Logs {
		if entry.Action != "user.password_set" || entry.Actor.ID != adminID {
			t.Errorf("entry %+v outside the filter", entry)
		}
	}

	if resp, _ := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/audit-logs/filters", "", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("filters without a token: status = %d, want 401", resp.StatusCode)
	}
}
