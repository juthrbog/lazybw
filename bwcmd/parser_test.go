package bwcmd

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    VaultStatus
		wantErr bool
	}{
		{
			name:  "locked",
			input: []byte(`{"status":"locked","userEmail":"u@t.com","lastSync":""}`),
			want:  VaultStatus{Status: "locked", UserEmail: "u@t.com"},
		},
		{
			name:  "unauthenticated",
			input: []byte(`{"status":"unauthenticated","userEmail":"","lastSync":""}`),
			want:  VaultStatus{Status: "unauthenticated"},
		},
		{
			name:  "unlocked with sync",
			input: []byte(`{"status":"unlocked","userEmail":"u@t.com","lastSync":"2024-01-01T00:00:00Z"}`),
			want:  VaultStatus{Status: "unlocked", UserEmail: "u@t.com", LastSync: "2024-01-01T00:00:00Z"},
		},
		{
			name:    "malformed JSON",
			input:   []byte(`{bad`),
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "null",
			input:   []byte("null\n"),
			wantErr: true,
		},
		{
			name:    "unknown status",
			input:   []byte(`{"status":"pending","userEmail":"u@t.com"}`),
			wantErr: true,
		},
		{
			name:    "no status",
			input:   []byte(`{}`),
			wantErr: true,
		},
		{
			name:  "byte order mark and white space",
			input: []byte("\n\xEF\xBB\xBF  {\"status\":\"locked\",\"userEmail\":\"u@t.com\"}\r\n"),
			want:  VaultStatus{Status: "locked", UserEmail: "u@t.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStatus(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.UserEmail != tt.want.UserEmail {
				t.Errorf("UserEmail = %q, want %q", got.UserEmail, tt.want.UserEmail)
			}
			if got.LastSync != tt.want.LastSync {
				t.Errorf("LastSync = %q, want %q", got.LastSync, tt.want.LastSync)
			}
		})
	}
}

func TestParseItems(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantCount int
		wantType  ItemType
		wantErr   bool
	}{
		{
			name:      "login item",
			input:     []byte(`[{"id":"1","type":1,"name":"Gmail","login":{"username":"u@t.com","password":"pw","totp":"","uris":[]}}]`),
			wantCount: 1,
			wantType:  ItemTypeLogin,
		},
		{
			name:      "card item",
			input:     []byte(`[{"id":"2","type":3,"name":"Visa","card":{"cardholderName":"John","number":"4242424242424242","expMonth":"12","expYear":"27","code":"123"}}]`),
			wantCount: 1,
			wantType:  ItemTypeCard,
		},
		{
			name:      "secure note",
			input:     []byte(`[{"id":"3","type":2,"name":"Keys","notes":"secret","secureNote":{"type":0}}]`),
			wantCount: 1,
			wantType:  ItemTypeSecureNote,
		},
		{
			name:      "multiple items",
			input:     []byte(`[{"id":"1","type":1,"name":"A"},{"id":"2","type":3,"name":"B"},{"id":"3","type":2,"name":"C"}]`),
			wantCount: 3,
		},
		{
			name:      "empty array",
			input:     []byte(`[]`),
			wantCount: 0,
		},
		{
			name:    "malformed JSON",
			input:   []byte(`{bad`),
			wantErr: true,
		},
		{
			name:      "byte order mark",
			input:     []byte("\xEF\xBB\xBF[{\"id\":\"1\",\"type\":1,\"name\":\"A\"}]\n"),
			wantCount: 1,
			wantType:  ItemTypeLogin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseItems(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("got %d items, want %d", len(got), tt.wantCount)
			}
			if tt.wantType != 0 && len(got) > 0 && got[0].Type != tt.wantType {
				t.Errorf("Type = %d, want %d", got[0].Type, tt.wantType)
			}
		})
	}
}

func TestParseErrorsNameTheCommand(t *testing.T) {
	statusErr := func(in string) error { _, err := ParseStatus([]byte(in)); return err }
	itemsErr := func(in string) error { _, err := ParseItems([]byte(in)); return err }
	tests := []struct {
		name   string
		err    error
		prefix string
		json   bool // the error wraps the encoding/json error
	}{
		{"empty status", statusErr(""), "parse bw status output: ", true},
		{"malformed status", statusErr("{bad"), "parse bw status output: ", true},
		{"null status", statusErr("null"), "parse bw status output: ", false},
		{"unknown status", statusErr(`{"status":"pending"}`), "parse bw status output: ", false},
		{"empty items", itemsErr(""), "parse bw list items output: ", true},
		{"items not a list", itemsErr(`{"id":"1"}`), "parse bw list items output: ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.HasPrefix(tt.err.Error(), tt.prefix) {
				t.Errorf("err = %q, want prefix %q", tt.err, tt.prefix)
			}
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			wrapped := errors.As(tt.err, &syntaxErr) || errors.As(tt.err, &typeErr)
			if wrapped != tt.json {
				t.Errorf("wraps a json error = %v, want %v", wrapped, tt.json)
			}
		})
	}
}
