// Package validation implements SPEC-GO.md §5.4's request validation: an
// English per-field message table, built on go-playground/validator/v10 for
// the individual rule checks (validator.Var, one tag at a time) but with its
// own sequencing — stop evaluating a field's remaining rules only once
// `required`/`accepted` fails, otherwise collect every failing rule — which
// validator's own struct-tag chains don't provide.
//
// Handlers decode request bodies into map[string]any (not typed structs), so
// a field present with the wrong JSON type (e.g. a number where a string was
// expected) can be reported with its own message instead of a generic JSON
// decode error (SPEC-GO.md §5.4's "string"/"integer"/"boolean"/"array" rules
// only make sense against a loosely-typed value).
package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	govalidator "github.com/go-playground/validator/v10"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

var engine = govalidator.New()

// Validator accumulates per-field messages across a single request's fields.
type Validator struct {
	errors map[string][]string
}

func New() *Validator {
	return &Validator{errors: map[string][]string{}}
}

// Valid reports whether every field validated so far passed.
func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

// Errors returns the accumulated messages in the exact shape response.Fail's
// messages parameter expects, or nil if there were none.
func (v *Validator) Errors() map[string][]string {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

func (v *Validator) addError(attribute, message string) {
	v.errors[attribute] = append(v.errors[attribute], message)
}

// Field starts a rule chain for attribute (e.g. "phone_number", or
// "facilities.0" for an array item) with the given decoded JSON value (nil if
// the key was absent, consistent with map[string]any lookups).
func (v *Validator) Field(attribute string, value any) *FieldValidator {
	return &FieldValidator{v: v, attribute: attribute, value: value}
}

// FieldValidator runs rules for a single field, in the order they're called.
type FieldValidator struct {
	v         *Validator
	attribute string
	value     any
	stopped   bool // required/accepted failed: skip every rule after it
	skipped   bool // Optional() saw an absent/empty value: skip every rule
}

func (f *FieldValidator) skip() bool { return f.stopped || f.skipped }

func (f *FieldValidator) fail(message string, stopping bool) *FieldValidator {
	f.v.addError(f.attribute, f.message(message))
	if stopping {
		f.stopped = true
	}
	return f
}

// message substitutes :attribute (SPEC-GO.md §5.4: underscores become spaces)
// into the English template.
func (f *FieldValidator) message(template string) string {
	label := strings.ReplaceAll(f.attribute, "_", " ")
	return strings.ReplaceAll(template, ":attribute", label)
}

// isEmpty reports whether value counts as "not sent/empty" for Optional's
// skip-entirely behavior (SPEC-GO.md §5.4).
func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return s == ""
	}
	return false
}

// Optional skips every subsequent rule for this field when the value is
// absent/empty, without recording an error either way.
func (f *FieldValidator) Optional() *FieldValidator {
	if isEmpty(f.value) {
		f.skipped = true
	}
	return f
}

// Required fails if the value is absent/empty; failing it stops the chain.
func (f *FieldValidator) Required() *FieldValidator {
	if f.skip() {
		return f
	}
	if isEmpty(f.value) {
		return f.fail("The :attribute field is required.", true)
	}
	return f
}

// Accepted fails unless the value is boolean true; failing it stops the chain
// (used for e.g. is_agree_privacy_policy).
func (f *FieldValidator) Accepted() *FieldValidator {
	if f.skip() {
		return f
	}
	if b, ok := f.value.(bool); !ok || !b {
		return f.fail("The :attribute field must be accepted.", true)
	}
	return f
}

func (f *FieldValidator) String() *FieldValidator {
	if f.skip() {
		return f
	}
	if _, ok := f.value.(string); !ok {
		f.fail("The :attribute field must be a string.", false)
	}
	return f
}

// isWholeNumber reports whether value is a JSON number with no fractional
// part (JSON decodes every number as float64, so "integer" can't just be a
// Go type switch on int).
func isWholeNumber(value any) (float64, bool) {
	n, ok := value.(float64)
	if !ok {
		return 0, false
	}
	return n, n == float64(int64(n))
}

func (f *FieldValidator) Integer() *FieldValidator {
	if f.skip() {
		return f
	}
	if _, ok := isWholeNumber(f.value); !ok {
		f.fail("The :attribute field must be an integer.", false)
	}
	return f
}

func (f *FieldValidator) Numeric() *FieldValidator {
	if f.skip() {
		return f
	}
	if _, ok := f.value.(float64); !ok {
		f.fail("The :attribute field must be a number.", false)
	}
	return f
}

func (f *FieldValidator) Boolean() *FieldValidator {
	if f.skip() {
		return f
	}
	if _, ok := f.value.(bool); !ok {
		f.fail("The :attribute field must be true or false.", false)
	}
	return f
}

func (f *FieldValidator) Array() *FieldValidator {
	if f.skip() {
		return f
	}
	if _, ok := f.value.([]any); !ok {
		f.fail("The :attribute field must be an array.", false)
	}
	return f
}

func (f *FieldValidator) Email() *FieldValidator {
	if f.skip() {
		return f
	}
	s, ok := f.value.(string)
	if !ok || engine.Var(s, "email") != nil {
		f.fail("The :attribute field must be a valid email address.", false)
	}
	return f
}

// MinStringLen enforces a minimum string length ("min (string)" in SPEC-GO.md §5.4).
func (f *FieldValidator) MinStringLen(min int) *FieldValidator {
	if f.skip() {
		return f
	}
	s, ok := f.value.(string)
	if !ok || len([]rune(s)) < min {
		f.fail(fmt.Sprintf("The :attribute field must be at least %d characters.", min), false)
	}
	return f
}

// MaxStringLen enforces a maximum string length ("max (string)").
func (f *FieldValidator) MaxStringLen(max int) *FieldValidator {
	if f.skip() {
		return f
	}
	s, ok := f.value.(string)
	if !ok || len([]rune(s)) > max {
		f.fail(fmt.Sprintf("The :attribute field must not be greater than %d characters.", max), false)
	}
	return f
}

// MinNumber enforces a numeric lower bound ("min (angka)").
func (f *FieldValidator) MinNumber(min float64) *FieldValidator {
	if f.skip() {
		return f
	}
	n, ok := f.value.(float64)
	if !ok || n < min {
		f.fail(fmt.Sprintf("The :attribute field must be at least %s.", trimTrailingZeros(min)), false)
	}
	return f
}

// BetweenNumber enforces an inclusive numeric range ("between (angka)").
func (f *FieldValidator) BetweenNumber(min, max float64) *FieldValidator {
	if f.skip() {
		return f
	}
	n, ok := f.value.(float64)
	if !ok || n < min || n > max {
		f.fail(fmt.Sprintf("The :attribute field must be between %s and %s.", trimTrailingZeros(min), trimTrailingZeros(max)), false)
	}
	return f
}

// MaxArrayLen enforces a maximum item count ("max (array)").
func (f *FieldValidator) MaxArrayLen(max int) *FieldValidator {
	if f.skip() {
		return f
	}
	arr, ok := f.value.([]any)
	if !ok || len(arr) > max {
		f.fail(fmt.Sprintf("The :attribute field must not have more than %d items.", max), false)
	}
	return f
}

// Regex enforces a regular expression match ("regex").
func (f *FieldValidator) Regex(pattern *regexp.Regexp) *FieldValidator {
	if f.skip() {
		return f
	}
	s, ok := f.value.(string)
	if !ok || !pattern.MatchString(s) {
		f.fail("The :attribute field format is invalid.", false)
	}
	return f
}

// In enforces membership in allowed ("in"/"exists" share one message per
// SPEC-GO.md §5.4; In covers the literal-list case).
func (f *FieldValidator) In(allowed ...string) *FieldValidator {
	if f.skip() {
		return f
	}
	s, ok := f.value.(string)
	valid := ok
	if valid {
		valid = false
		for _, a := range allowed {
			if s == a {
				valid = true
				break
			}
		}
	}
	if !valid {
		f.fail("The selected :attribute is invalid.", false)
	}
	return f
}

// Exists enforces that found (computed by the caller, typically a DB lookup)
// is true. Same message as In (SPEC-GO.md §5.4: "in / exists").
func (f *FieldValidator) Exists(found bool) *FieldValidator {
	if f.skip() {
		return f
	}
	if !found {
		f.fail("The selected :attribute is invalid.", false)
	}
	return f
}

// Unique enforces that taken (computed by the caller, typically a DB lookup)
// is false.
func (f *FieldValidator) Unique(taken bool) *FieldValidator {
	if f.skip() {
		return f
	}
	if taken {
		f.fail("The :attribute has already been taken.", false)
	}
	return f
}

const dateLayout = "2006-01-02"

// Date enforces "YYYY-MM-DD" and returns the parsed date in the WIB zone so
// AfterOrEqualToday can compare it directly; ok is false if Date itself failed.
func (f *FieldValidator) Date() (*FieldValidator, time.Time, bool) {
	if f.skip() {
		return f, time.Time{}, false
	}
	s, ok := f.value.(string)
	if ok {
		if t, err := time.ParseInLocation(dateLayout, s, money.WIB); err == nil {
			return f, t, true
		}
	}
	f.fail("The :attribute field must be a valid date.", false)
	return f, time.Time{}, false
}

// AfterOrEqualToday enforces parsed >= today in WIB; call only after Date
// succeeded (SPEC-GO.md: "after_or_equal:today").
func (f *FieldValidator) AfterOrEqualToday(parsed time.Time) *FieldValidator {
	if f.skip() {
		return f
	}
	if parsed.Before(money.TodayWIB()) {
		f.fail("The :attribute field must be a date after or equal to today.", false)
	}
	return f
}

func trimTrailingZeros(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}
