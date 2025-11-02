package slack

import (
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header http.Header
		want   http.Header
	}{
		{
			name:   "nil header",
			header: nil,
			want:   nil,
		},
		{
			name: "no sensitive headers",
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Custom":     []string{"value"},
			},
			want: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Custom":     []string{"value"},
			},
		},
		{
			name: "authorization header redacted",
			header: http.Header{
				"Authorization": []string{"Bearer secret-token"},
			},
			want: http.Header{
				"Authorization": []string{"Bearer [redacted]"},
			},
		},
		{
			name: "authorization header non bearer",
			header: http.Header{
				"Authorization": []string{"Basic abcd"},
			},
			want: http.Header{
				"Authorization": []string{"[redacted]"},
			},
		},
		{
			name: "mixed case header key",
			header: http.Header{
				"authOrization": []string{"Bearer secret"},
			},
			want: http.Header{
				"authOrization": []string{"Bearer [redacted]"},
			},
		},
		{
			name: "multiple authorization values",
			header: http.Header{
				"Authorization": []string{"Bearer token", "Basic foo"},
			},
			want: http.Header{
				"Authorization": []string{"Bearer [redacted]", "[redacted]"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := sanitizeHeaders(tc.header)

			if tc.want == nil {
				if got != nil {
					t.Fatalf("expected nil header, got: %#v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected non-nil header")
			}

			if len(got) != len(tc.want) {
				t.Fatalf("header length mismatch: got %d, want %d", len(got), len(tc.want))
			}

			for key, wantValues := range tc.want {
				gotValues, ok := got[key]
				if !ok {
					t.Fatalf("expected key %q not found", key)
				}

				if len(gotValues) != len(wantValues) {
					t.Fatalf("values length mismatch for key %q: got %d, want %d", key, len(gotValues), len(wantValues))
				}

				for i := range gotValues {
					if gotValues[i] != wantValues[i] {
						t.Fatalf("value mismatch for key %q at index %d: got %q, want %q", key, i, gotValues[i], wantValues[i])
					}
				}
			}

		})
	}
}

func TestMaskSensitiveValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "bearer token", in: "Bearer secret", want: "Bearer [redacted]"},
		{name: "case insensitive bearer", in: "bearer secret", want: "Bearer [redacted]"},
		{name: "other scheme", in: "Basic secret", want: "[redacted]"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := maskSensitiveValue(tc.in)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	t.Parallel()

	if !isSensitiveHeader("Authorization") {
		t.Fatal("expected Authorization to be sensitive")
	}

	if !isSensitiveHeader("authorization") {
		t.Fatal("expected lowercase authorization to be sensitive")
	}

	if isSensitiveHeader("Content-Type") {
		t.Fatal("did not expect Content-Type to be sensitive")
	}
}

func TestSanitizeHeadersClonesInput(t *testing.T) {
	t.Parallel()

	header := http.Header{"Authorization": []string{"Bearer secret"}}

	got := sanitizeHeaders(header)

	if strings.Contains(header.Get("Authorization"), "[redacted]") {
		t.Fatal("expected original header to remain unchanged")
	}

	got.Set("Authorization", "Bearer mutated")
	if header.Get("Authorization") != "Bearer secret" {
		t.Fatal("expected original header to remain unchanged after mutating sanitized header")
	}
}
