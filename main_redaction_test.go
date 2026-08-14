package main

import (
	"strings"
	"testing"
)

// Both credentials genuinely travel in the URL: the API key may be passed as
// ?key=, and the console WebSocket must send its JWT as ?token= because a
// browser can't set headers on a WS handshake. These lock in that neither
// ever reaches a log line intact.

func TestCredentialRedaction(t *testing.T) {
	cases := []struct {
		name    string
		path    string
		want    string
		secrets []string
	}{
		{
			name:    "api key as a query param",
			path:    "/api/admin/invitations?key=supersecret123",
			want:    "/api/admin/invitations?key=REDACTED",
			secrets: []string{"supersecret123"},
		},
		{
			name:    "jwt on the console websocket",
			path:    "/api/console?token=eyJhbGciOiJIUzI1NiJ9.payload.signature",
			want:    "/api/console?token=REDACTED",
			secrets: []string{"eyJhbGciOiJIUzI1NiJ9.payload.signature"},
		},
		{
			name:    "credential after another param",
			path:    "/api/files?path=world&key=supersecret123",
			want:    "/api/files?path=world&key=REDACTED",
			secrets: []string{"supersecret123"},
		},
		{
			name:    "both credentials at once",
			path:    "/api/console?token=jwtvalue&key=keyvalue",
			want:    "/api/console?token=REDACTED&key=REDACTED",
			secrets: []string{"jwtvalue", "keyvalue"},
		},
		{
			name: "a following param survives redaction",
			path: "/api/files?key=supersecret123&path=world",
			want: "/api/files?key=REDACTED&path=world",
			// The redaction must stop at '&' rather than eating the rest of
			// the query, or the log loses the information it exists for.
			secrets: []string{"supersecret123"},
		},
		{
			name:    "no credentials, untouched",
			path:    "/api/players",
			want:    "/api/players",
			secrets: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := credentialParamRe.ReplaceAllString(tc.path, "${1}REDACTED")
			if got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
			for _, secret := range tc.secrets {
				if strings.Contains(got, secret) {
					t.Errorf("secret %q survived redaction in %q", secret, got)
				}
			}
		})
	}
}
