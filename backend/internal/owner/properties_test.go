package owner

import "testing"

func TestPropertySlug(t *testing.T) {
	cases := map[string]string{
		"Kost Asti Beji":         "kost-asti-beji-7",
		"  Kost -- Asti!! Beji ": "kost-asti-beji-7",
		"Kost Café Margonda":     "kost-caf-margonda-7",
		"!!!":                    "7",
	}
	for name, want := range cases {
		if got := propertySlug(name, 7); got != want {
			t.Errorf("propertySlug(%q) = %q, want %q", name, got, want)
		}
	}
}
