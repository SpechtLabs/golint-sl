package credname

import (
	"slices"
	"testing"
)

func TestNameWords(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "password", want: []string{"password"}},
		{name: "dbPassword", want: []string{"db", "password"}},
		{name: "api_key", want: []string{"api", "key"}},
		{name: "API_KEY", want: []string{"api", "key"}},
		{name: "APIKey", want: []string{"api", "key"}},
		{name: "dbPASSWORD", want: []string{"db", "password"}},
		{name: "oauth2Token", want: []string{"oauth2", "token"}},
		{name: "secretB64", want: []string{"secret", "b64"}},
		{name: "tls-client-secret", want: []string{"tls", "client", "secret"}},
		{name: "__token__", want: []string{"token"}},
		{name: "HTTPServer", want: []string{"http", "server"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Words(tt.name); !slices.Equal(got, tt.want) {
				t.Errorf("Words(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsCredential(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "password", want: true},
		{name: "dbPassword", want: true},
		{name: "authToken", want: true},
		{name: "api_key", want: true},
		{name: "APIKey", want: true},
		{name: "privateKey", want: true},
		{name: "SECRET_KEY", want: true},
		{name: "clientSecret", want: true},
		{name: "apikey", want: true},
		{name: "tokenStr", want: true},
		{name: "basicAuth", want: true},
		{name: "credentials", want: true},
		{name: "cred", want: true},
		{name: "awsCreds", want: true},
		{name: "credit", want: false},
		{name: "author", want: false},
		{name: "tokenizerMode", want: false},
		{name: "tokenURL", want: false},
		{name: "passwordPolicy", want: false},
		{name: "secretName", want: false},
		{name: "monkey", want: false},
		{name: "key", want: false},
		{name: "cacheKey", want: false},
		{name: "value", want: false},
		{name: "_", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCredential(tt.name); got != tt.want {
				t.Errorf("IsCredential(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
