package shared

import "testing"

func TestJSONOrNull(t *testing.T) {
	if got := JSONOrNull(nil); got != "null" {
		t.Errorf("JSONOrNull(nil) = %q, want %q", got, "null")
	}

	type sample struct {
		Name string `json:"name"`
	}
	got := JSONOrNull(sample{Name: "test"})
	want := `{"name":"test"}`
	if got != want {
		t.Errorf("JSONOrNull(struct) = %q, want %q", got, want)
	}

	// An empty string must never be produced — MySQL's JSON column type
	// rejects it (error 3140), which is exactly the bug this function fixes.
	if got := JSONOrNull(nil); got == "" {
		t.Error("JSONOrNull must never return an empty string")
	}
}
