package auth

import (
	"strings"
	"testing"

	"aura/internal/domain"
)

func TestPasswordHash(t *testing.T) {
	Iterations = 1000
	h, err := HashPassword("aura-mx1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "pbkdf2-sha256$1000$") {
		t.Fatalf("hash = %q", h)
	}
	h2, _ := HashPassword("aura-mx1")
	if h == h2 {
		t.Error("two hashes of one password are equal: salt is not random")
	}

	tests := []struct {
		name     string
		hash     string
		password string
		want     bool
		wantErr  bool
	}{
		{"right password", h, "aura-mx1", true, false},
		{"wrong password", h, "aura-mx2", false, false},
		{"empty password", h, "", false, false},
		{"other scheme", "bcrypt$10$x$y", "aura-mx1", false, true},
		{"garbage", "nope", "aura-mx1", false, true},
		{"bad iterations", "pbkdf2-sha256$0$AAAA$AAAA", "aura-mx1", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckPassword(tt.hash, tt.password)
			if got != tt.want || (err != nil) != tt.wantErr {
				t.Errorf("CheckPassword = %v, %v; want %v, err %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestTokens(t *testing.T) {
	a, _ := NewToken()
	b, _ := NewToken()
	if a == b || len(a) < 40 {
		t.Fatalf("tokens %q %q", a, b)
	}
	if HashToken(a) == a || HashToken(a) != HashToken(a) || len(HashToken(a)) != 64 {
		t.Error("HashToken must be a stable SHA-256 hex digest")
	}
}

func TestAccessPolicy(t *testing.T) {
	amina := domain.Astronaut{ID: 4, Role: domain.RoleCrew}
	lucas := domain.Astronaut{ID: 5, Role: domain.RoleCrew}
	alex := domain.Astronaut{ID: 2, Role: domain.RoleCommander}
	kenji := domain.Astronaut{ID: 3, Role: domain.RoleDeputyCommander}
	sarah := domain.Astronaut{ID: 1, Role: domain.RoleFlightSurgeon}

	tests := []struct {
		name           string
		viewer, target domain.Astronaut
		view, log      bool
	}{
		{"crew sees self", amina, amina, true, true},
		{"crew cannot see others", amina, lucas, false, false},
		{"crew cannot see commander", amina, alex, false, false},
		{"commander sees crew, logs only self", alex, amina, true, false},
		{"commander logs self", alex, alex, true, true},
		{"deputy sees crew", kenji, lucas, true, false},
		{"surgeon sees crew", sarah, lucas, true, false},
		{"surgeon has nothing to log", sarah, sarah, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanView(tt.viewer, tt.target); got != tt.view {
				t.Errorf("CanView = %v, want %v", got, tt.view)
			}
			if got := CanLog(tt.viewer, tt.target); got != tt.log {
				t.Errorf("CanLog = %v, want %v", got, tt.log)
			}
		})
	}
}
