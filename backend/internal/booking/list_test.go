package booking

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

type recordingListStore struct {
	baseStubStore
	rows     []dbgen.ListBookingsForUserRow
	captured *dbgen.ListBookingsForUserParams
}

func (s recordingListStore) ListBookingsForUser(ctx context.Context, arg dbgen.ListBookingsForUserParams) ([]dbgen.ListBookingsForUserRow, error) {
	*s.captured = arg
	return s.rows, nil
}

func TestListHandlerClampsLimitAndDefaultsPage(t *testing.T) {
	var gotArg dbgen.ListBookingsForUserParams
	deps := Deps{Store: recordingListStore{captured: &gotArg}}

	req := httptest.NewRequest(http.MethodGet, "/booking/my?limit=999&page=2", nil)
	w := httptest.NewRecorder()
	ListHandler(deps, authctx.User{UserID: 1})(w, req)

	if gotArg.ResultLimit != maxLimit {
		t.Errorf("ResultLimit = %d, want %d", gotArg.ResultLimit, maxLimit)
	}
	if gotArg.ResultOffset != maxLimit {
		t.Errorf("ResultOffset = %d, want %d (page 2)", gotArg.ResultOffset, maxLimit)
	}

	var body struct {
		CurrentPage int `json:"current_page"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body.CurrentPage != 2 {
		t.Errorf("current_page = %d, want 2", body.CurrentPage)
	}
}
