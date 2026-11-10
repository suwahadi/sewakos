package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeverity(t *testing.T) {
	cases := map[int]string{200: "OK", 201: "OK", 400: "WARNING", 422: "WARNING", 404: "WARNING", 500: "ERROR", 502: "ERROR"}
	for status, want := range cases {
		if got := Severity(status); got != want {
			t.Errorf("Severity(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestSuccessEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	Success(w, http.StatusOK, "Login berhasil.", map[string]any{"token": "abc"})

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != true {
		t.Errorf("status = %v, want true", body["status"])
	}
	if body["token"] != "abc" {
		t.Errorf("token = %v, want abc", body["token"])
	}
	meta := body["meta"].(map[string]any)
	if meta["message"] != "Login berhasil." || meta["severity"] != "OK" {
		t.Errorf("meta = %+v", meta)
	}
}

func TestFailWithNullMessages(t *testing.T) {
	w := httptest.NewRecorder()
	FailWithNullMessages(w, http.StatusUnauthorized, MsgLoginFailed)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	raw, ok := body["messages"]
	if !ok {
		t.Fatal("messages key must be present")
	}
	if raw != nil {
		t.Errorf("messages = %v, want null", raw)
	}
}

func TestFailOmitsMessagesWhenNil(t *testing.T) {
	w := httptest.NewRecorder()
	Fail(w, http.StatusNotFound, MsgNotFound, nil)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := body["messages"]; ok {
		t.Error("messages key must be absent, not null")
	}
}
