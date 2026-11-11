package validation

import (
	"regexp"
	"testing"
)

func firstMsg(t *testing.T, v *Validator, attribute string) string {
	t.Helper()
	msgs := v.Errors()[attribute]
	if len(msgs) == 0 {
		t.Fatalf("no error recorded for %q (errors=%v)", attribute, v.Errors())
	}
	return msgs[0]
}

func TestRequired(t *testing.T) {
	v := New()
	v.Field("name", nil).Required()
	if got := firstMsg(t, v, "name"); got != "The name field is required." {
		t.Errorf("got %q", got)
	}
}

func TestRequiredStopsFurtherRules(t *testing.T) {
	v := New()
	v.Field("phone_number", nil).Required().MinStringLen(8).MaxStringLen(20)
	msgs := v.Errors()["phone_number"]
	if len(msgs) != 1 {
		t.Fatalf("expected exactly one message after required fails, got %v", msgs)
	}
}

func TestAcceptedStopsFurtherRules(t *testing.T) {
	v := New()
	v.Field("is_agree_privacy_policy", false).Accepted().Boolean()
	msgs := v.Errors()["is_agree_privacy_policy"]
	if len(msgs) != 1 || msgs[0] != "The is agree privacy policy field must be accepted." {
		t.Errorf("got %v", msgs)
	}
}

func TestNonRequiredFailuresAllAccumulate(t *testing.T) {
	v := New()
	// min and max both genuinely fail together (value shorter than min, over max item count N/A here);
	// use a string that's both too short to hit min and matches a failing regex, to prove both report.
	v.Field("code", "ab").Required().MinStringLen(5).Regex(regexp.MustCompile(`^\d+$`))
	msgs := v.Errors()["code"]
	if len(msgs) != 2 {
		t.Fatalf("expected both non-stopping rule failures reported, got %v", msgs)
	}
}

func TestOptionalSkipsEntirelyWhenAbsent(t *testing.T) {
	v := New()
	v.Field("email", nil).Optional().Email()
	if !v.Valid() {
		t.Errorf("optional+absent should produce no errors, got %v", v.Errors())
	}
}

func TestOptionalStillValidatesWhenPresent(t *testing.T) {
	v := New()
	v.Field("email", "not-an-email").Optional().Email()
	if got := firstMsg(t, v, "email"); got != "The email field must be a valid email address." {
		t.Errorf("got %q", got)
	}
}

func TestStringRule(t *testing.T) {
	v := New()
	v.Field("name", float64(5)).String()
	if got := firstMsg(t, v, "name"); got != "The name field must be a string." {
		t.Errorf("got %q", got)
	}
}

func TestIntegerRuleRejectsFraction(t *testing.T) {
	v := New()
	v.Field("duration", 1.5).Integer()
	if got := firstMsg(t, v, "duration"); got != "The duration field must be an integer." {
		t.Errorf("got %q", got)
	}
	v2 := New()
	v2.Field("duration", float64(3)).Integer()
	if !v2.Valid() {
		t.Errorf("whole-number float should pass Integer(), got %v", v2.Errors())
	}
}

func TestBooleanRule(t *testing.T) {
	v := New()
	v.Field("flag", "true").Boolean()
	if got := firstMsg(t, v, "flag"); got != "The flag field must be true or false." {
		t.Errorf("got %q", got)
	}
}

func TestArrayRule(t *testing.T) {
	v := New()
	v.Field("facilities", "not-an-array").Array()
	if got := firstMsg(t, v, "facilities"); got != "The facilities field must be an array." {
		t.Errorf("got %q", got)
	}
}

func TestEmailRule(t *testing.T) {
	v := New()
	v.Field("email", "bad").Email()
	if got := firstMsg(t, v, "email"); got != "The email field must be a valid email address." {
		t.Errorf("got %q", got)
	}
}

func TestMinMaxStringLen(t *testing.T) {
	v := New()
	v.Field("password", "short").MinStringLen(8)
	if got := firstMsg(t, v, "password"); got != "The password field must be at least 8 characters." {
		t.Errorf("got %q", got)
	}

	v2 := New()
	v2.Field("name", "this is way too long a name surely").MaxStringLen(10)
	if got := firstMsg(t, v2, "name"); got != "The name field must not be greater than 10 characters." {
		t.Errorf("got %q", got)
	}
}

func TestRegexRule(t *testing.T) {
	v := New()
	v.Field("phone_number", "0812").Regex(regexp.MustCompile(`^08\d{7,12}$`))
	if got := firstMsg(t, v, "phone_number"); got != "The phone number field format is invalid." {
		t.Errorf("got %q", got)
	}
}

func TestInAndExistsShareMessage(t *testing.T) {
	v := New()
	v.Field("type", "apartemen").In("city", "campus", "mrt")
	if got := firstMsg(t, v, "type"); got != "The selected type is invalid." {
		t.Errorf("got %q", got)
	}

	v2 := New()
	v2.Field("area_id", float64(999)).Exists(false)
	if got := firstMsg(t, v2, "area_id"); got != "The selected area id is invalid." {
		t.Errorf("got %q", got)
	}
}

func TestUniqueRule(t *testing.T) {
	v := New()
	v.Field("phone_number", "081200000099").Unique(true)
	if got := firstMsg(t, v, "phone_number"); got != "The phone number has already been taken." {
		t.Errorf("got %q", got)
	}
}

func TestDateRule(t *testing.T) {
	v := New()
	fv, _, ok := v.Field("start_date", "not-a-date").Date()
	if ok {
		t.Fatal("expected Date() to fail")
	}
	_ = fv
	if got := firstMsg(t, v, "start_date"); got != "The start date field must be a valid date." {
		t.Errorf("got %q", got)
	}
}

func TestAfterOrEqualToday(t *testing.T) {
	v := New()
	_, parsed, ok := v.Field("start_date", "2000-01-01").Date()
	if !ok {
		t.Fatal("expected Date() to succeed")
	}
	v.Field("start_date", "2000-01-01").AfterOrEqualToday(parsed)
	if got := firstMsg(t, v, "start_date"); got != "The start date field must be a date after or equal to today." {
		t.Errorf("got %q", got)
	}
}

func TestNumericRule(t *testing.T) {
	v := New()
	v.Field("latitude", "not-numeric").Numeric()
	if got := firstMsg(t, v, "latitude"); got != "The latitude field must be a number." {
		t.Errorf("got %q", got)
	}
}

func TestMinNumberAndBetweenNumber(t *testing.T) {
	v := New()
	v.Field("available", float64(-1)).MinNumber(0)
	if got := firstMsg(t, v, "available"); got != "The available field must be at least 0." {
		t.Errorf("got %q", got)
	}

	v2 := New()
	v2.Field("latitude", float64(120)).BetweenNumber(-90, 90)
	if got := firstMsg(t, v2, "latitude"); got != "The latitude field must be between -90 and 90." {
		t.Errorf("got %q", got)
	}
}

func TestMaxArrayLen(t *testing.T) {
	v := New()
	v.Field("facilities", []any{"a", "b", "c"}).MaxArrayLen(2)
	if got := firstMsg(t, v, "facilities"); got != "The facilities field must not have more than 2 items." {
		t.Errorf("got %q", got)
	}
}

// TestDottedArrayItemAttribute proves a nested array item (e.g. facilities[0])
// is validated under its dotted attribute name, exactly as SPEC-GO.md §5.4's
// own example: "facilities.0: The facilities.0 field must not be greater than
// :max characters."
func TestDottedArrayItemAttribute(t *testing.T) {
	v := New()
	v.Field("facilities.0", "this facility name is absolutely far too long for the limit").MaxStringLen(50)
	if got := firstMsg(t, v, "facilities.0"); got != "The facilities.0 field must not be greater than 50 characters." {
		t.Errorf("got %q", got)
	}
}

func TestValidPassesWithNoErrors(t *testing.T) {
	v := New()
	v.Field("name", "Test User").Required().String().MaxStringLen(255)
	v.Field("email", nil).Optional().Email()
	if !v.Valid() || v.Errors() != nil {
		t.Errorf("expected valid, got errors=%v", v.Errors())
	}
}

func TestDistinctNamesTheArrayItem(t *testing.T) {
	v := New()
	v.Field("monthly_duration_options.1", 3.0).Integer().Distinct(true)
	if got := firstMsg(t, v, "monthly_duration_options.1"); got != "The monthly duration options.1 field has a duplicate value." {
		t.Errorf("got %q", got)
	}
}
