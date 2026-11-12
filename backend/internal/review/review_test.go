package review

import "testing"

func TestMaskTenantNameKeepsFirstWordAndSecondInitial(t *testing.T) {
	cases := map[string]string{
		"Anak Kos Contoh": "Anak K.",
		"Budi":            "Budi",
		"  Siti   Aminah": "Siti A.",
		"":                "",
	}
	for name, want := range cases {
		if got := MaskTenantName(name); got != want {
			t.Errorf("MaskTenantName(%q) = %q, want %q", name, got, want)
		}
	}
}
