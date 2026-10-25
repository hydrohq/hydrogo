//go:build smoke

// Package smoke runs the real hydro binary against a throw-away project and
// checks what a person would otherwise have to check by hand before trusting a
// release: the security basics and that the API stays fast with real amounts
// of data. Run it with:
//
//	go test -tags smoke ./tests/smoke -count=1
//
// (go run ./tools/check does this for you.) Nothing here needs a browser,
// credentials, Docker or the network.
package smoke

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// host is the public origin the project is configured with. HydroGo answers
// 421 to any other Host header, so every request here sends it.
const host = "hydro.test"

type project struct {
	t       *testing.T
	dir     string
	bin     string
	config  string
	port    int
	baseURL string
	log     *bytes.Buffer
}

// start builds nothing: bin is the compiled binary. It scaffolds a minimal
// project, adds the given content types, starts the server and waits for it.
func start(t *testing.T, bin string, contentTypes map[string]string) *project {
	t.Helper()
	dir := t.TempDir()
	p := &project{t: t, dir: dir, bin: bin, log: &bytes.Buffer{}}
	p.run("Quartz-Meadow-4417-Lantern\n", "new", "proj", "--preset", "minimal", "--admin-email", "admin@smoke.test", "--admin-password-stdin")
	p.dir = filepath.Join(dir, "proj")
	p.port = freePort(t)
	p.baseURL = fmt.Sprintf("http://127.0.0.1:%d", p.port)
	p.config = filepath.Join(p.dir, "config.yaml")
	dialect, dsn := p.database()
	cfg := fmt.Sprintf(`profile: compact
modules: [core, auth, pages, media, site, collections]
http:
  address: "127.0.0.1:%d"
  public_origin: "http://%s"
database:
  dialect: %q
  dsn: %q
media:
  dir: "data/media"
content_types_dir: "content-types"
security:
  secure_cookies: false
  enable_hsts: false
`, p.port, host, dialect, dsn)
	if err := os.WriteFile(p.config, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range contentTypes {
		if err := os.WriteFile(filepath.Join(p.dir, "content-types", name+".yaml"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p.run("", "migrate", "--config", p.config)
	if dialect != "sqlite" {
		// `hydro new` made the administrator in its own SQLite file.
		p.run("Quartz-Meadow-4417-Lantern\n", "create-admin", "--email", "admin@smoke.test", "--password-stdin", "--config", p.config)
	}

	p.serveOn(p.config, p.port)
	return p
}

// database picks the engine the server runs on. SQLite by default;
// SMOKE_DIALECT=postgres or mysql (with HYDRO_TEST_POSTGRES_URL or
// HYDRO_TEST_MYSQL_URL pointing at a server you do not mind tests creating
// databases on) runs the same checks there, each test in a fresh database.
func (p *project) database() (dialect, dsn string) {
	p.t.Helper()
	switch d := os.Getenv("SMOKE_DIALECT"); d {
	case "", "sqlite":
		return "sqlite", "data/hydro.db"
	case "postgres":
		base := os.Getenv("HYDRO_TEST_POSTGRES_URL")
		if base == "" {
			p.t.Fatal("SMOKE_DIALECT=postgres needs HYDRO_TEST_POSTGRES_URL")
		}
		name := p.freshDatabase("pgx", base, "postgres")
		u, _ := url.Parse(base)
		u.Path = "/" + name
		return "postgres", u.String()
	case "mysql":
		base := os.Getenv("HYDRO_TEST_MYSQL_URL")
		if base == "" {
			p.t.Fatal("SMOKE_DIALECT=mysql needs HYDRO_TEST_MYSQL_URL")
		}
		name := p.freshDatabase("mysql", base, "mysql")
		cfg, err := mysql.ParseDSN(base)
		if err != nil {
			p.t.Fatal(err)
		}
		cfg.DBName = name
		return "mysql", cfg.FormatDSN()
	default:
		p.t.Fatalf("unknown SMOKE_DIALECT %q", d)
		return "", ""
	}
}

func (p *project) freshDatabase(driver, base, kind string) string {
	db, err := sql.Open(driver, base)
	if err != nil {
		p.t.Fatal(err)
	}
	name := fmt.Sprintf("smoke_%d_%d", os.Getpid(), time.Now().UnixNano()%1_000_000_000)
	if _, err := db.Exec("CREATE DATABASE " + name); err != nil {
		p.t.Fatalf("create database: %v", err)
	}
	p.t.Cleanup(func() {
		_, _ = db.Exec("DROP DATABASE IF EXISTS " + name)
		db.Close()
	})
	return name
}

func (p *project) run(stdin string, args ...string) string {
	p.t.Helper()
	cmd := exec.Command(p.bin, args...)
	cmd.Dir = p.dir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		p.t.Fatalf("hydro %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// serviceAccount creates an API principal and returns its bearer token and
// the e-mail HydroGo knows it by (what row access compares against).
func (p *project) serviceAccount(name string, permissions ...string) (token, email string) {
	p.t.Helper()
	out := p.run("", "service-account", "create", "--name", name, "--permissions", strings.Join(permissions, ","), "--expires-days", "30", "--config", p.config)
	m := regexp.MustCompile(`(?m)^(hyd_\S+)$`).FindStringSubmatch(out)
	if m == nil {
		p.t.Fatalf("no token in:\n%s", out)
	}
	return m[1], "service+" + name + "@hydrogo.invalid"
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) json() map[string]any {
	var v map[string]any
	_ = json.Unmarshal(r.body, &v)
	return v
}

func (r response) data() map[string]any {
	d, _ := r.json()["data"].(map[string]any)
	return d
}

func (r response) list() []any {
	d, _ := r.json()["data"].([]any)
	return d
}

type opts struct {
	token   string
	body    any    // JSON-encoded unless it is a string or []byte
	ctype   string // overrides the content type
	header  map[string]string
	client  *http.Client
	noHost  bool
	rawHost string
}

func (p *project) do(method, path string, o *opts) (response, error) {
	if o == nil {
		o = &opts{}
	}
	var rd io.Reader
	ctype := o.ctype
	switch b := o.body.(type) {
	case nil:
	case string:
		rd = strings.NewReader(b)
	case []byte:
		rd = bytes.NewReader(b)
	default:
		raw, _ := json.Marshal(b)
		rd = bytes.NewReader(raw)
		if ctype == "" {
			ctype = "application/json"
		}
	}
	req, err := http.NewRequestWithContext(context.Background(), method, p.baseURL+path, rd)
	if err != nil {
		return response{}, err
	}
	switch {
	case o.rawHost != "":
		req.Host = o.rawHost
	case !o.noHost:
		req.Host = host
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	if o.token != "" {
		req.Header.Set("Authorization", "Bearer "+o.token)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range o.header {
		req.Header.Set(k, v)
	}
	hc := o.client
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, err := hc.Do(req)
	if err != nil {
		return response{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return response{status: res.StatusCode, header: res.Header, body: raw}, nil
}

func (p *project) must(method, path string, o *opts) response {
	p.t.Helper()
	res, err := p.do(method, path, o)
	if err != nil {
		p.t.Fatalf("%s %s: %v", method, path, err)
	}
	return res
}

// session is a browser-like client: a cookie jar, the CSRF token and the
// signed-in user, the way the admin panel talks to the server.
type session struct {
	p    *project
	hc   *http.Client
	csrf string
}

func (p *project) login(email, password string) (*session, response) {
	p.t.Helper()
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	tok := p.must("GET", "/csrf-token", &opts{client: hc})
	csrf, _ := tok.json()["token"].(string)
	if csrf == "" {
		csrf, _ = tok.data()["token"].(string)
	}
	res := p.must("POST", "/auth/login", &opts{client: hc, ctype: "application/x-www-form-urlencoded", body: url.Values{"email": {email}, "password": {password}}.Encode(), header: map[string]string{"X-CSRF-Token": csrf}})
	s := &session{p: p, hc: hc}
	if res.status < 300 {
		me := p.must("GET", "/api/v1/admin/me", &opts{client: hc})
		s.csrf, _ = me.data()["csrf_token"].(string)
	}
	return s, res
}

// anonymous is a visitor with no account: it still needs the CSRF cookie and
// header, even for the public endpoints that set a password.
func (p *project) anonymous() *session {
	p.t.Helper()
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	tok := p.must("GET", "/csrf-token", &opts{client: hc})
	csrf, _ := tok.json()["token"].(string)
	if csrf == "" {
		csrf, _ = tok.data()["token"].(string)
	}
	return &session{p: p, hc: hc, csrf: csrf}
}

func (s *session) do(method, path string, o *opts) response {
	s.p.t.Helper()
	if o == nil {
		o = &opts{}
	}
	o.client = s.hc
	if o.header == nil {
		o.header = map[string]string{}
	}
	if _, ok := o.header["X-CSRF-Token"]; !ok && method != "GET" {
		o.header["X-CSRF-Token"] = s.csrf
	}
	return s.p.must(method, path, o)
}

// stats

func percentile(d []time.Duration, q float64) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), d...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	i := int(float64(len(s)-1) * q)
	return s[i]
}

// serveOn starts a server for the given config and waits until it answers; the
// returned project talks to it. The first call is start()'s own server.
func (p *project) serveOn(config string, port int) *project {
	t := p.t
	t.Helper()
	q := &project{t: t, dir: p.dir, bin: p.bin, config: config, port: port, baseURL: fmt.Sprintf("http://127.0.0.1:%d", port), log: &bytes.Buffer{}}
	cmd := exec.Command(q.bin, "serve", "--config", config)
	cmd.Dir = q.dir
	cmd.Stdout, cmd.Stderr = q.log, q.log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		if t.Failed() {
			tail := q.log.String()
			if len(tail) > 2500 {
				tail = tail[len(tail)-2500:]
			}
			t.Logf("server log tail (%s):\n%s", filepath.Base(config), tail)
		}
	})
	deadline := time.Now().Add(30 * time.Second)
	for {
		if res, err := q.do("GET", "/healthz", nil); err == nil && res.status == 200 {
			return q
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not become ready:\n%s", q.log.String())
		}
		time.Sleep(150 * time.Millisecond)
	}
}
