package contract

import (
	"regexp"
	"testing"
)

// recordingT lets these tests assert on AssertJSONMatches's pass/fail outcome
// without actually failing the outer test when a negative case is exercised.
type recordingT struct {
	failed bool
}

func (r *recordingT) Errorf(format string, args ...any) { r.failed = true }
func (r *recordingT) Fatalf(format string, args ...any) { r.failed = true }
func (r *recordingT) Helper()                           {}

func check(t *testing.T, got, want string, dynamic map[string]*regexp.Regexp) bool {
	t.Helper()
	rt := &recordingT{}
	AssertJSONMatches(rt, []byte(got), []byte(want), dynamic)
	return !rt.failed
}

func TestKeyOrderIgnored(t *testing.T) {
	if !check(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`, nil) {
		t.Error("key order should not matter")
	}
}

func TestNullVsMissingVsEmptyAreDistinct(t *testing.T) {
	if check(t, `{"messages":null}`, `{}`, nil) {
		t.Error("explicit null must not match an absent key")
	}
	if check(t, `{}`, `{"messages":null}`, nil) {
		t.Error("an absent key must not match an explicit null")
	}
	if check(t, `{"photos":null}`, `{"photos":[]}`, nil) {
		t.Error("null must not match []")
	}
	if check(t, `{"photos":[]}`, `{"photos":{}}`, nil) {
		t.Error("[] must not match {}")
	}
}

func TestTypeMismatchFails(t *testing.T) {
	if check(t, `{"id":"2"}`, `{"id":2}`, nil) {
		t.Error("string \"2\" must not match number 2")
	}
}

func TestUnexpectedKeyFails(t *testing.T) {
	if check(t, `{"a":1,"extra":true}`, `{"a":1}`, nil) {
		t.Error("a key not in the fixture must fail")
	}
}

func TestArrayOrderAndLengthMatter(t *testing.T) {
	if check(t, `{"a":[1,2]}`, `{"a":[2,1]}`, nil) {
		t.Error("array element order must matter")
	}
	if check(t, `{"a":[1]}`, `{"a":[1,2]}`, nil) {
		t.Error("array length mismatch must fail")
	}
}

func TestDynamicFieldAppliesRegex(t *testing.T) {
	dynamic := map[string]*regexp.Regexp{"$.token": regexp.MustCompile(`^\d+\|[A-Za-z0-9]{48}$`)}
	validPlain := "2|" + repeat("a", 48)
	if !check(t, `{"token":"`+validPlain+`"}`, `{"token":"IGNORED-BY-REGEX"}`, dynamic) {
		t.Error("a token matching the dynamic regex should pass regardless of the fixture's literal value")
	}
	if check(t, `{"token":"2|too-short"}`, `{"token":"IGNORED-BY-REGEX"}`, dynamic) {
		t.Error("a token not matching the dynamic regex should fail")
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func TestNestedPathMatching(t *testing.T) {
	got := `{"user":{"id":2,"name":"Anak Kos Contoh","role":"tenant"}}`
	want := `{"user":{"id":2,"name":"Anak Kos Contoh","role":"tenant"}}`
	if !check(t, got, want, nil) {
		t.Error("identical nested objects should match")
	}
}
