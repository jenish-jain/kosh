package sheets_test

import (
	"testing"

	"kosh/sheets"
)

// ── ColStr ────────────────────────────────────────────────────────────────────

func TestColStr(t *testing.T) {
	tests := []struct {
		name string
		row  []interface{}
		col  int
		want string
	}{
		{
			name: "empty row returns empty string",
			row:  []interface{}{},
			col:  0,
			want: "",
		},
		{
			name: "out-of-bounds index returns empty string",
			row:  []interface{}{"a", "b"},
			col:  5,
			want: "",
		},
		{
			name: "normal string value",
			row:  []interface{}{"hello", "world"},
			col:  0,
			want: "hello",
		},
		{
			name: "second column",
			row:  []interface{}{"hello", "world"},
			col:  1,
			want: "world",
		},
		{
			name: "integer value is stringified",
			row:  []interface{}{42},
			col:  0,
			want: "42",
		},
		{
			name: "float value is stringified",
			row:  []interface{}{3.14},
			col:  0,
			want: "3.14",
		},
		{
			name: "nil value at in-bounds index",
			row:  []interface{}{nil},
			col:  0,
			want: "<nil>",
		},
		{
			name: "empty string value",
			row:  []interface{}{""},
			col:  0,
			want: "",
		},
		{
			name: "index exactly at length boundary returns empty string",
			row:  []interface{}{"only"},
			col:  1, // len == 1, col == 1: out of bounds
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sheets.ColStr(tc.row, tc.col)
			if got != tc.want {
				t.Errorf("ColStr(%v, %d) = %q, want %q", tc.row, tc.col, got, tc.want)
			}
		})
	}
}

// ── ColFloat ──────────────────────────────────────────────────────────────────

func TestColFloat(t *testing.T) {
	tests := []struct {
		name string
		row  []interface{}
		col  int
		want float64
	}{
		{
			name: "empty row returns 0",
			row:  []interface{}{},
			col:  0,
			want: 0,
		},
		{
			name: "out-of-bounds index returns 0",
			row:  []interface{}{"1.5"},
			col:  9,
			want: 0,
		},
		{
			name: "normal float string",
			row:  []interface{}{"3.14"},
			col:  0,
			want: 3.14,
		},
		{
			name: "integer string",
			row:  []interface{}{"42"},
			col:  0,
			want: 42,
		},
		{
			name: "zero value",
			row:  []interface{}{"0"},
			col:  0,
			want: 0,
		},
		{
			name: "negative value",
			row:  []interface{}{"-7.5"},
			col:  0,
			want: -7.5,
		},
		{
			name: "string with commas (Indian/US formatting)",
			row:  []interface{}{"1,23,456.78"},
			col:  0,
			want: 123456.78,
		},
		{
			name: "string with commas (thousands separator)",
			row:  []interface{}{"1,000,000"},
			col:  0,
			want: 1000000,
		},
		{
			name: "non-numeric string returns 0",
			row:  []interface{}{"not-a-number"},
			col:  0,
			want: 0,
		},
		{
			name: "empty string returns 0",
			row:  []interface{}{""},
			col:  0,
			want: 0,
		},
		{
			name: "numeric value stored as float64 (not string)",
			row:  []interface{}{float64(9.99)},
			col:  0,
			want: 9.99,
		},
		{
			name: "large positive value",
			row:  []interface{}{"9999999.99"},
			col:  0,
			want: 9999999.99,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sheets.ColFloat(tc.row, tc.col)
			if got != tc.want {
				t.Errorf("ColFloat(%v, %d) = %v, want %v", tc.row, tc.col, got, tc.want)
			}
		})
	}
}

// ── ColInt ────────────────────────────────────────────────────────────────────

func TestColInt(t *testing.T) {
	tests := []struct {
		name string
		row  []interface{}
		col  int
		want int
	}{
		{
			name: "empty row returns 0",
			row:  []interface{}{},
			col:  0,
			want: 0,
		},
		{
			name: "out-of-bounds index returns 0",
			row:  []interface{}{"5"},
			col:  3,
			want: 0,
		},
		{
			name: "normal integer string",
			row:  []interface{}{"7"},
			col:  0,
			want: 7,
		},
		{
			name: "zero value",
			row:  []interface{}{"0"},
			col:  0,
			want: 0,
		},
		{
			name: "negative integer",
			row:  []interface{}{"-3"},
			col:  0,
			want: -3,
		},
		{
			name: "float string is truncated (not rounded)",
			row:  []interface{}{"9.9"},
			col:  0,
			want: 9,
		},
		{
			name: "non-numeric string returns 0",
			row:  []interface{}{"abc"},
			col:  0,
			want: 0,
		},
		{
			name: "string with commas stripped before parse",
			row:  []interface{}{"1,000"},
			col:  0,
			want: 1000,
		},
		{
			name: "large integer",
			row:  []interface{}{"1000000"},
			col:  0,
			want: 1000000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sheets.ColInt(tc.row, tc.col)
			if got != tc.want {
				t.Errorf("ColInt(%v, %d) = %d, want %d", tc.row, tc.col, got, tc.want)
			}
		})
	}
}

// ── EnvOrDefault ──────────────────────────────────────────────────────────────

func TestEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		envValue string // empty means "do not set the var"
		def      string
		want     string
	}{
		{
			name:     "env var set returns env value",
			key:      "TEST_KOSH_VAR",
			envValue: "from-env",
			def:      "default-val",
			want:     "from-env",
		},
		{
			name:     "env var not set returns default",
			key:      "TEST_KOSH_MISSING",
			envValue: "",
			def:      "fallback",
			want:     "fallback",
		},
		{
			name:     "empty string env var treated as unset, returns default",
			key:      "TEST_KOSH_EMPTY",
			envValue: "", // t.Setenv will set to "", EnvOrDefault should return def
			def:      "use-default",
			want:     "use-default",
		},
		{
			name:     "default is empty string when env not set",
			key:      "TEST_KOSH_NO_DEFAULT",
			envValue: "",
			def:      "",
			want:     "",
		},
		{
			name:     "env var with whitespace is returned as-is",
			key:      "TEST_KOSH_SPACES",
			envValue: "  spaced  ",
			def:      "default",
			want:     "  spaced  ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				t.Setenv(tc.key, tc.envValue)
			}
			got := sheets.EnvOrDefault(tc.key, tc.def)
			if got != tc.want {
				t.Errorf("EnvOrDefault(%q, %q) = %q, want %q", tc.key, tc.def, got, tc.want)
			}
		})
	}
}

// Explicit test: env var that IS set overrides a non-empty default.
func TestEnvOrDefault_EnvOverridesNonEmptyDefault(t *testing.T) {
	const key = "TEST_KOSH_OVERRIDE"
	t.Setenv(key, "winner")
	got := sheets.EnvOrDefault(key, "loser")
	if got != "winner" {
		t.Errorf("expected %q, got %q", "winner", got)
	}
}
