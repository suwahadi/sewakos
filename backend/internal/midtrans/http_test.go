package midtrans

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *HTTPClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewHTTPClient("test-server-key", false, srv.URL)
}

func TestCreateTransactionSendsAuthAndBody(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody TransactionRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(TransactionResponse{Token: "tok", RedirectURL: "https://example.com/redirect"})
	})

	resp, err := c.CreateTransaction(t.Context(), TransactionRequest{
		TransactionDetails: TransactionDetails{OrderID: "KOS-1-1", GrossAmount: 100000},
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if resp.Token != "tok" || resp.RedirectURL != "https://example.com/redirect" {
		t.Errorf("response = %+v", resp)
	}
	if gotPath != "/snap/v1/transactions" {
		t.Errorf("path = %q", gotPath)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("test-server-key:"))
	if gotAuth != wantAuth {
		t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
	}
	if gotBody.TransactionDetails.OrderID != "KOS-1-1" || gotBody.TransactionDetails.GrossAmount != 100000 {
		t.Errorf("request body = %+v", gotBody)
	}
}

func TestGetStatusParsesResponse(t *testing.T) {
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(StatusResponse{
			OrderID: "KOS-1-1", TransactionStatus: "settlement", GrossAmount: "100000.00",
		})
	})

	resp, err := c.GetStatus(t.Context(), "KOS-1-1")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if gotPath != "/v2/KOS-1-1/status" {
		t.Errorf("path = %q", gotPath)
	}
	if resp.TransactionStatus != "settlement" {
		t.Errorf("TransactionStatus = %q", resp.TransactionStatus)
	}
}

func TestExpireTransactionPostsToExpireEndpoint(t *testing.T) {
	var gotMethod, gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	if err := c.ExpireTransaction(t.Context(), "KOS-1-1"); err != nil {
		t.Fatalf("ExpireTransaction: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v2/KOS-1-1/expire" {
		t.Errorf("method/path = %s %s", gotMethod, gotPath)
	}
}

func TestNonSuccessStatusIsAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status_message":"boom"}`))
	})

	if _, err := c.CreateTransaction(t.Context(), TransactionRequest{}); err == nil {
		t.Fatal("CreateTransaction err = nil, want error for 500 response")
	}
}
