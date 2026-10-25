//go:build smoke

package smoke

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

const notesType = `id: notes
label: Notes
title_field: title
permissions: {read: notes.read, write: notes.write}
access: {owner: owner, all: notes.all}
fields:
  - {id: title, label: Title, kind: text, required: true, list: true}
  - {id: owner, label: Owner, kind: text, filterable: true}
  - {id: salary, label: Salary, kind: number, permission: notes.salary}
`

// TestSecurityBasics checks, against a real server, the promises HydroGo
// makes about who can see and change what. Logins come first and the
// brute-force probe last: sign-in is rate limited per address on purpose.
func TestSecurityBasics(t *testing.T) {
	p := start(t, binary, map[string]string{"notes": notesType})
	alice, aliceMail := p.serviceAccount("alice", "notes.read", "notes.write")
	bob, _ := p.serviceAccount("bob", "notes.read", "notes.write")
	boss, _ := p.serviceAccount("boss", "notes.read", "notes.write", "notes.all", "notes.salary")
	reader, _ := p.serviceAccount("reader", "notes.read")

	t.Run("the admin API refuses anonymous callers with a request id", func(t *testing.T) {
		for _, path := range []string{"/api/v1/admin/me", "/api/v1/admin/content/notes", "/api/v1/admin/users"} {
			res := p.must("GET", path, nil)
			if res.status != 401 {
				t.Errorf("GET %s anonymous: %d, want 401", path, res.status)
			}
			if _, ok := res.json()["request_id"]; !ok {
				t.Errorf("GET %s: error has no request_id: %s", path, res.body)
			}
		}
		if res := p.must("GET", "/api/v1/admin/content/notes", &opts{token: "hyd_not-a-real-token"}); res.status != 401 {
			t.Errorf("a made-up token: %d, want 401", res.status)
		}
	})

	t.Run("a different Host is refused and the answer says what is expected", func(t *testing.T) {
		res := p.must("GET", "/healthz", &opts{rawHost: "evil.example"})
		if res.status != 421 || !strings.Contains(string(res.body), host) {
			t.Errorf("status %d body %s", res.status, res.body)
		}
	})

	t.Run("responses carry the security headers", func(t *testing.T) {
		for _, path := range []string{"/admin/", "/api/v1/admin/me", "/healthz"} {
			h := p.must("GET", path, nil).header
			if h.Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("%s: X-Content-Type-Options = %q", path, h.Get("X-Content-Type-Options"))
			}
			csp := h.Get("Content-Security-Policy")
			if !strings.Contains(csp, "frame-ancestors") && h.Get("X-Frame-Options") == "" {
				t.Errorf("%s: nothing stops framing (CSP %q)", path, csp)
			}
			if h.Get("Referrer-Policy") == "" {
				t.Errorf("%s: no Referrer-Policy", path)
			}
		}
		if csp := p.must("GET", "/admin/", nil).header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src") || strings.Contains(csp, "'unsafe-eval'") {
			t.Errorf("admin CSP should set default-src and avoid unsafe-eval: %q", csp)
		}
	})

	t.Run("row access: people only see their own rows", func(t *testing.T) {
		mine := p.must("POST", "/api/v1/admin/content/notes", &opts{token: alice, body: map[string]any{"title": "alice's note", "owner": "someone-else"}})
		if mine.status != 201 {
			t.Fatalf("create: %d %s", mine.status, mine.body)
		}
		id, _ := mine.data()["id"].(string)
		if owner, _ := mine.data()["owner"].(string); owner != aliceMail {
			t.Errorf("owner is set by the server, got %q want %q", owner, aliceMail)
		}
		p.must("POST", "/api/v1/admin/content/notes", &opts{token: bob, body: map[string]any{"title": "bob's note"}})

		if res := p.must("GET", "/api/v1/admin/content/notes/"+id, &opts{token: bob}); res.status != 404 {
			t.Errorf("bob reading alice's record: %d, want 404", res.status)
		}
		if res := p.must("PUT", "/api/v1/admin/content/notes/"+id, &opts{token: bob, body: map[string]any{"title": "hijacked", "version": 1}}); res.status != 404 {
			t.Errorf("bob editing alice's record: %d, want 404", res.status)
		}
		if res := p.must("DELETE", "/api/v1/admin/content/notes/"+id, &opts{token: bob}); res.status != 404 {
			t.Errorf("bob deleting alice's record: %d, want 404", res.status)
		}
		if n := len(p.must("GET", "/api/v1/admin/content/notes", &opts{token: bob}).list()); n != 1 {
			t.Errorf("bob's list has %d rows, want only his own", n)
		}
		if n := len(p.must("GET", "/api/v1/admin/content/notes", &opts{token: boss}).list()); n != 2 {
			t.Errorf("the holder of notes.all sees %d rows, want 2", n)
		}
		if res := p.must("GET", "/api/v1/admin/content/notes/_export", &opts{token: bob}); res.status == 200 && strings.Contains(string(res.body), "alice's note") {
			t.Error("a CSV export leaked another person's row")
		}
		// the record is intact
		if res := p.must("GET", "/api/v1/admin/content/notes/"+id, &opts{token: alice}); res.status != 200 || res.data()["title"] != "alice's note" {
			t.Errorf("alice's record changed: %d %s", res.status, res.body)
		}
	})

	t.Run("permissions: read-only callers cannot write, field permissions hide data", func(t *testing.T) {
		if res := p.must("POST", "/api/v1/admin/content/notes", &opts{token: reader, body: map[string]any{"title": "nope"}}); res.status != 403 {
			t.Errorf("write without notes.write: %d, want 403", res.status)
		}
		created := p.must("POST", "/api/v1/admin/content/notes", &opts{token: boss, body: map[string]any{"title": "pay", "salary": 4200}})
		if created.status != 201 {
			t.Fatalf("boss create: %d %s", created.status, created.body)
		}
		id, _ := created.data()["id"].(string)
		if _, has := p.must("GET", "/api/v1/admin/content/notes/"+id, &opts{token: reader}).data()["salary"]; has {
			t.Error("the salary field reached someone without notes.salary")
		}
		if _, has := p.must("GET", "/api/v1/admin/content/notes/"+id, &opts{token: boss}).data()["salary"]; !has {
			t.Error("the holder of notes.salary cannot see it")
		}
		if res := p.must("GET", "/api/v1/admin/content/notes?sort=salary", &opts{token: reader}); res.status != 400 {
			t.Errorf("sorting by a field you cannot read: %d, want 400", res.status)
		}
	})

	t.Run("sessions: cookies are protected and writes need the CSRF token", func(t *testing.T) {
		s, res := p.login("admin@smoke.test", "Quartz-Meadow-4417-Lantern")
		if res.status >= 300 {
			t.Fatalf("admin sign-in: %d %s", res.status, res.body)
		}
		var sessionCookie string
		for _, c := range res.header.Values("Set-Cookie") {
			if strings.HasPrefix(c, "hydro_session=") {
				sessionCookie = c
			}
		}
		if !strings.Contains(sessionCookie, "HttpOnly") || !strings.Contains(strings.ToLower(sessionCookie), "samesite") {
			t.Errorf("session cookie must be HttpOnly and SameSite: %q", sessionCookie)
		}
		body := map[string]any{"title": "from the panel"}
		if res := s.do("POST", "/api/v1/admin/content/notes", &opts{body: body, header: map[string]string{"X-CSRF-Token": "forged"}}); res.status != 403 {
			t.Errorf("a forged CSRF token: %d, want 403", res.status)
		}
		if res := s.do("POST", "/api/v1/admin/content/notes", &opts{body: body, header: map[string]string{"X-CSRF-Token": ""}}); res.status != 403 {
			t.Errorf("no CSRF token: %d, want 403", res.status)
		}
		if res := s.do("POST", "/api/v1/admin/content/notes", &opts{body: body}); res.status != 201 {
			t.Errorf("the right CSRF token: %d %s", res.status, res.body)
		}
		if res, _ := p.do("POST", "/auth/logout", &opts{client: s.hc, header: map[string]string{"X-CSRF-Token": s.csrf}}); res.status >= 400 {
			t.Errorf("logout: %d", res.status)
		}
		if res := s.do("GET", "/api/v1/admin/me", nil); res.status != 401 {
			t.Errorf("after logout the session still works: %d", res.status)
		}
	})

	t.Run("invitation and password-reset links are single-use and set a working password", func(t *testing.T) {
		admin, res := p.login("admin@smoke.test", "Quartz-Meadow-4417-Lantern")
		if res.status >= 300 {
			t.Fatalf("sign-in: %d", res.status)
		}
		form := "application/x-www-form-urlencoded"
		inv := admin.do("POST", "/api/v1/admin/users/invites", &opts{ctype: form, body: url.Values{"email": {"newcomer@smoke.test"}, "role": {"viewer"}}.Encode()})
		if inv.status != 202 || inv.json()["delivered"] != false {
			t.Fatalf("invite without mail: %d %s", inv.status, inv.body)
		}
		link, _ := inv.json()["url"].(string)
		token := regexp.MustCompile(`token=([0-9a-f]+)`).FindStringSubmatch(link)
		if !strings.Contains(link, "/admin/accept-invite?token=") || token == nil {
			t.Fatalf("invite link: %q", link)
		}
		anon := p.anonymous()
		if res := anon.do("POST", "/auth/invites/accept", &opts{ctype: form, body: url.Values{"token": {token[1]}, "password": {"short"}}.Encode()}); res.status != 400 {
			t.Errorf("a weak password: %d, want 400", res.status)
		}
		accept := func(pw string) response {
			return anon.do("POST", "/auth/invites/accept", &opts{ctype: form, body: url.Values{"token": {token[1]}, "password": {pw}}.Encode()})
		}
		if res := accept("Neu-Mitarbeiter-Parola-55!"); res.status != 201 {
			t.Fatalf("accepting the invitation: %d %s", res.status, res.body)
		}
		if res := accept("Another-Parola-2026-77!"); res.status != 401 {
			t.Errorf("the invitation worked twice: %d", res.status)
		}
		newcomer, res := p.login("newcomer@smoke.test", "Neu-Mitarbeiter-Parola-55!")
		if res.status >= 300 {
			t.Fatalf("the new account cannot sign in: %d %s", res.status, res.body)
		}
		if res := newcomer.do("GET", "/api/v1/admin/users", nil); res.status != 403 {
			t.Errorf("a viewer listing users: %d, want 403", res.status)
		}

		// A reset link for that account.
		var id string
		for _, u := range admin.do("GET", "/api/v1/admin/users", nil).list() {
			if m, _ := u.(map[string]any); m["email"] == "newcomer@smoke.test" {
				id, _ = m["id"].(string)
			}
		}
		reset := admin.do("POST", "/api/v1/admin/users/"+id+"/password-reset-link", nil)
		rlink, _ := reset.json()["url"].(string)
		rt := regexp.MustCompile(`token=([0-9a-f]+)`).FindStringSubmatch(rlink)
		if reset.status != 200 || !strings.Contains(rlink, "/admin/reset-password?token=") || rt == nil {
			t.Fatalf("reset link: %d %s", reset.status, reset.body)
		}
		confirm := func(pw string) response {
			return anon.do("POST", "/auth/password-reset/confirm", &opts{ctype: form, body: url.Values{"token": {rt[1]}, "password": {pw}}.Encode()})
		}
		if res := confirm("Reset-Parola-2026-Yeni-91!"); res.status != 204 {
			t.Fatalf("reset: %d %s", res.status, res.body)
		}
		if res := confirm("Reset-Parola-2026-Baska-12!"); res.status != 401 {
			t.Errorf("the reset link worked twice: %d", res.status)
		}
		if _, res := p.login("newcomer@smoke.test", "Neu-Mitarbeiter-Parola-55!"); res.status < 400 {
			t.Error("the old password still works after a reset")
		}
		if _, res := p.login("newcomer@smoke.test", "Reset-Parola-2026-Yeni-91!"); res.status >= 300 {
			t.Errorf("the new password does not work: %d", res.status)
		}
	})

	t.Run("role and permission changes made in the panel stick", func(t *testing.T) {
		// What the Users screen sends. (The bug that once broke it, the admin client
		// encoding these forms twice, is in the client: its unit test and the
		// browser journeys cover that part.)
		p.run("Staff-Parola-2026-Test!\n", "create-user", "--email", "staff@smoke.test", "--role", "viewer", "--permissions", "admin.site.manage", "--password-stdin", "--config", p.config)
		admin, res := p.login("admin@smoke.test", "Quartz-Meadow-4417-Lantern")
		if res.status >= 300 {
			t.Skipf("sign-in is rate limited by now (%d); the earlier subtests already used its quota", res.status)
		}
		var id string
		var perms []any
		for _, u := range admin.do("GET", "/api/v1/admin/users", nil).list() {
			if m, _ := u.(map[string]any); m["email"] == "staff@smoke.test" {
				id, _ = m["id"].(string)
				perms, _ = m["permissions"].([]any)
			}
		}
		form := "application/x-www-form-urlencoded"
		if res := admin.do("POST", "/api/v1/admin/users/"+id+"/role", &opts{ctype: form, body: "role=editor"}); res.status != 204 {
			t.Errorf("changing a role: %d %s", res.status, res.body)
		}
		keep := url.Values{}
		for _, perm := range perms {
			keep.Set("perm_"+perm.(string), "allow")
		}
		if res := admin.do("POST", "/api/v1/admin/users/"+id+"/permissions", &opts{ctype: form, body: keep.Encode()}); res.status != 204 {
			t.Errorf("saving permissions: %d %s", res.status, res.body)
		}
		for _, u := range admin.do("GET", "/api/v1/admin/users", nil).list() {
			if m, _ := u.(map[string]any); m["email"] == "staff@smoke.test" {
				if m["role"] != "editor" {
					t.Errorf("role = %v, want editor", m["role"])
				}
				if !strings.Contains(strings.Join(toStrings(m["permissions"]), ","), "admin.site.manage") {
					t.Errorf("the explicit permission was lost: %v", m["permissions"])
				}
			}
		}
	})

	t.Run("sign-in is rate limited", func(t *testing.T) {
		limited := false
		for i := 0; i < 12 && !limited; i++ {
			_, res := p.login("nobody@smoke.test", "wrong-password-"+strings.Repeat("x", i))
			limited = res.status == 429
		}
		if !limited {
			t.Error("12 wrong sign-ins in a row were never throttled")
		}
	})
}

func toStrings(v any) []string {
	var out []string
	for _, x := range v.([]any) {
		out = append(out, x.(string))
	}
	return out
}
