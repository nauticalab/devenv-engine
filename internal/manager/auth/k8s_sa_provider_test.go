package auth

import (
	"testing"
)

func TestParseServiceAccountUsername(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		wantSAName    string
		wantNamespace string
		wantErr       bool
	}{
		{
			name:          "valid service account",
			username:      "system:serviceaccount:devenv:devenv-testuser",
			wantSAName:    "devenv-testuser",
			wantNamespace: "devenv",
			wantErr:       false,
		},
		{
			name:          "service account with hyphen",
			username:      "system:serviceaccount:devenv:devenv-test-user",
			wantSAName:    "devenv-test-user",
			wantNamespace: "devenv",
			wantErr:       false,
		},
		{
			name:          "different namespace",
			username:      "system:serviceaccount:custom-ns:my-sa",
			wantSAName:    "my-sa",
			wantNamespace: "custom-ns",
			wantErr:       false,
		},
		{
			name:     "non-service account",
			username: "user@example.com",
			wantErr:  true,
		},
		{
			name:     "malformed - missing parts",
			username: "system:serviceaccount:devenv",
			wantErr:  true,
		},
		{
			name:     "empty username",
			username: "",
			wantErr:  true,
		},
		{
			name:     "wrong prefix",
			username: "system:user:test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saName, namespace, err := parseServiceAccountUsername(tt.username)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseServiceAccountUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if saName != tt.wantSAName {
					t.Errorf("saName = %v, want %v", saName, tt.wantSAName)
				}
				if namespace != tt.wantNamespace {
					t.Errorf("namespace = %v, want %v", namespace, tt.wantNamespace)
				}
			}
		})
	}
}

func TestParseDeveloperFromServiceAccount(t *testing.T) {
	tests := []struct {
		name    string
		saName  string
		pattern string
		wantDev string
		wantErr bool
	}{
		{
			name:    "valid devenv service account",
			saName:  "devenv-testuser",
			pattern: "devenv-{developer}",
			wantDev: "testuser",
			wantErr: false,
		},
		{
			name:    "developer name with hyphen",
			saName:  "devenv-test-user",
			pattern: "devenv-{developer}",
			wantDev: "test-user",
			wantErr: false,
		},
		{
			name:    "custom pattern",
			saName:  "myapp-john-doe",
			pattern: "myapp-{developer}",
			wantDev: "john-doe",
			wantErr: false,
		},
		{
			name:    "wrong prefix",
			saName:  "other-testuser",
			pattern: "devenv-{developer}",
			wantErr: true,
		},
		{
			name:    "manager service account",
			saName:  "devenv-manager",
			pattern: "devenv-{developer}",
			wantErr: true,
		},
		{
			name:    "empty developer name",
			saName:  "devenv-",
			pattern: "devenv-{developer}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dev, err := ParseDeveloperFromServiceAccount(tt.saName, tt.pattern)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDeveloperFromServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && dev != tt.wantDev {
				t.Errorf("developer = %v, want %v", dev, tt.wantDev)
			}
		})
	}
}

func TestSplitTwo(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{
			name: "normal split",
			s:    "namespace:saname",
			sep:  ":",
			want: []string{"namespace", "saname"},
		},
		{
			name: "multiple separators - splits on first",
			s:    "system:serviceaccount:devenv",
			sep:  ":",
			want: []string{"system", "serviceaccount:devenv"},
		},
		{
			name: "no separator",
			s:    "noseparator",
			sep:  ":",
			want: []string{"noseparator"},
		},
		{
			name: "empty string",
			s:    "",
			sep:  ":",
			want: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTwo(tt.s, tt.sep)
			if len(got) != len(tt.want) {
				t.Errorf("splitTwo() returned %d parts, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitTwo() part %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
