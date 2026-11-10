// Package contract runs the real HTTP API (built the same way cmd/api does)
// against a seeded database and compares its JSON responses semantically to
// the fixtures drawn from the contract docs (SPEC-GO.md §11, §1): key order
// and byte-for-byte formatting may differ, but every key, its type, and
// `null` vs. an absent key vs. `[]`/`{}` must match exactly.
package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// TestingT is the subset of *testing.T (or *testing.B) AssertJSONMatches
// needs; a small interface so callers can wrap it, e.g. to assert on the
// pass/fail outcome itself in compare_test.go.
type TestingT interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// AssertJSONMatches compares got against want (both raw JSON documents).
// dynamic maps a dot-path (e.g. "token", "verification_code.can_resend_at")
// to a regexp its value (formatted with %v) must match instead of being
// compared for exact equality — for fields a fixture can't pin down, like a
// random token or a timestamp.
func AssertJSONMatches(t TestingT, got, want []byte, dynamic map[string]*regexp.Regexp) {
	t.Helper()

	var gotVal, wantVal any
	if err := json.Unmarshal(got, &gotVal); err != nil {
		t.Fatalf("response is not valid JSON: %v\nbody: %s", err, got)
	}
	if err := json.Unmarshal(want, &wantVal); err != nil {
		t.Fatalf("fixture is not valid JSON: %v", err)
	}

	diffs := compareValue("$", gotVal, wantVal, dynamic)
	if len(diffs) > 0 {
		t.Errorf("response does not match fixture:\n%s\n\ngot body: %s", joinLines(diffs), got)
	}
}

func compareValue(path string, got, want any, dynamic map[string]*regexp.Regexp) []string {
	if re, ok := dynamic[path]; ok {
		if got == nil {
			return []string{fmt.Sprintf("%s: got null, want match of /%s/", path, re)}
		}
		if !re.MatchString(fmt.Sprintf("%v", got)) {
			return []string{fmt.Sprintf("%s: got %v, want match of /%s/", path, got, re)}
		}
		return nil
	}

	switch wantTyped := want.(type) {
	case nil:
		if got != nil {
			return []string{fmt.Sprintf("%s: got %v, want null", path, got)}
		}
		return nil
	case map[string]any:
		gotTyped, ok := got.(map[string]any)
		if !ok {
			return []string{fmt.Sprintf("%s: got %s, want object", path, typeName(got))}
		}
		var diffs []string
		for k, wv := range wantTyped {
			gv, present := gotTyped[k]
			if !present {
				diffs = append(diffs, fmt.Sprintf("%s.%s: missing key", path, k))
				continue
			}
			diffs = append(diffs, compareValue(path+"."+k, gv, wv, dynamic)...)
		}
		for k := range gotTyped {
			if _, ok := wantTyped[k]; !ok {
				diffs = append(diffs, fmt.Sprintf("%s.%s: unexpected key not in contract", path, k))
			}
		}
		return diffs
	case []any:
		gotTyped, ok := got.([]any)
		if !ok {
			return []string{fmt.Sprintf("%s: got %s, want array", path, typeName(got))}
		}
		if len(gotTyped) != len(wantTyped) {
			return []string{fmt.Sprintf("%s: got %d items, want %d", path, len(gotTyped), len(wantTyped))}
		}
		var diffs []string
		for i := range wantTyped {
			diffs = append(diffs, compareValue(fmt.Sprintf("%s[%d]", path, i), gotTyped[i], wantTyped[i], dynamic)...)
		}
		return diffs
	case string:
		gotTyped, ok := got.(string)
		if !ok || gotTyped != wantTyped {
			return []string{fmt.Sprintf("%s: got %s, want %q", path, repr(got), wantTyped)}
		}
		return nil
	case float64:
		gotTyped, ok := got.(float64)
		if !ok || gotTyped != wantTyped {
			return []string{fmt.Sprintf("%s: got %s, want %v", path, repr(got), wantTyped)}
		}
		return nil
	case bool:
		gotTyped, ok := got.(bool)
		if !ok || gotTyped != wantTyped {
			return []string{fmt.Sprintf("%s: got %s, want %v", path, repr(got), wantTyped)}
		}
		return nil
	default:
		return []string{fmt.Sprintf("%s: unsupported fixture value type %T", path, want)}
	}
}

func typeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func repr(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%v (%s)", v, typeName(v))
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += "- " + l
	}
	return out
}
