package main

import "testing"

func TestAuthHeader(t *testing.T) {
	client := &HomebridgeClient{}
	if got := client.authHeader(); got != "" {
		t.Fatalf("expected empty header, got %q", got)
	}

	client.token = &TokenCache{AccessToken: "abc", TokenType: ""}
	if got := client.authHeader(); got != "Bearer abc" {
		t.Fatalf("expected bearer header, got %q", got)
	}

	client.token = &TokenCache{AccessToken: "xyz", TokenType: "Token"}
	if got := client.authHeader(); got != "Token xyz" {
		t.Fatalf("expected token header, got %q", got)
	}
}

func TestValueToBool(t *testing.T) {
	cases := []struct {
		name  string
		input interface{}
		want  bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"float64 zero", float64(0), false},
		{"float64 one", float64(1), true},
		{"int zero", 0, false},
		{"int one", 1, true},
		{"int64 zero", int64(0), false},
		{"int64 one", int64(1), true},
		{"string empty", "", false},
		{"string zero", "0", false},
		{"string false", "false", false},
		{"string true", "true", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := valueToBool(tc.input); got != tc.want {
				t.Fatalf("valueToBool(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
