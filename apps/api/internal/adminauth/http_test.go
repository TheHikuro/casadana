package adminauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newRouter(svc *Service, cookieSecure bool) http.Handler {
	r := chi.NewRouter()
	Mount(r, svc, cookieSecure)
	return r
}

func loginAndGetJar(t *testing.T, baseURL, email, password string) http.CookieJar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}
	body := `{"email":"` + email + `","password":"` + password + `"}`
	resp, err := client.Post(baseURL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	return jar
}

func TestLogin_SetsCookie(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	body := `{"email":"loan@casa-dana.com","password":"correcthorse"}`
	resp, err := http.Post(srv.URL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var found *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookieName {
			found = c
		}
	}
	if found == nil {
		t.Fatal("session cookie was not set")
	}
	if !found.HttpOnly || !found.Secure {
		t.Errorf("cookie attrs HttpOnly=%v Secure=%v, want both true", found.HttpOnly, found.Secure)
	}
}

func TestLogin_WrongPassword_401(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	body := `{"email":"loan@casa-dana.com","password":"wrong"}`
	resp, err := http.Post(srv.URL+"/api/admin/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMe_WithoutCookie_401(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestMe_WithValidCookie_200(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	client := &http.Client{Jar: loginAndGetJar(t, srv.URL, "loan@casa-dana.com", "correcthorse")}
	resp, err := client.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out adminUserDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Email != "loan@casa-dana.com" {
		t.Errorf("email = %q", out.Email)
	}
}

func TestLogout_ClearsCookie(t *testing.T) {
	repo := &fakeRepo{}
	seedUser(t, repo, "admin-1", "loan@casa-dana.com", "correcthorse", d("2026-07-02"))
	svc := NewService(repo, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	client := &http.Client{Jar: loginAndGetJar(t, srv.URL, "loan@casa-dana.com", "correcthorse")}

	resp, err := client.Post(srv.URL+"/api/admin/logout", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	meResp, err := client.Get(srv.URL + "/api/admin/me")
	if err != nil {
		t.Fatal(err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status after logout = %d, want 401", meResp.StatusCode)
	}
}

func TestUsers_RequiresAuth(t *testing.T) {
	svc := NewService(&fakeRepo{}, fixedClock{t: d("2026-07-02")}, "test-secret")
	srv := httptest.NewServer(newRouter(svc, true))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/admin/users")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
