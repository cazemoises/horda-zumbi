package domain

import "testing"

func TestIsValidCharacterID(t *testing.T) {
	cases := []struct {
		name string
		id   CharacterID
		want bool
	}{
		{"tank is valid", CharacterTank, true},
		{"medic is valid", CharacterMedic, true},
		{"scout is valid", CharacterScout, true},
		{"unknown id is invalid", CharacterID("wizard"), false},
		{"empty id is invalid", CharacterID(""), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidCharacterID(tc.id); got != tc.want {
				t.Errorf("IsValidCharacterID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
