// Command side-service is a starting point for the code that does not belong
// in HydroGo: a service HydroGo asks before it saves a change (hooks) and
// that it calls on a schedule. It uses only the standard library and checks
// the request signature before it believes anything.
//
//	HOOK_SECRET=... go run .            # listens on :9000
//
// Wire it up in HydroGo's config.yaml:
//
//	hooks:
//	  - id: ledger
//	    url: http://127.0.0.1:9000/hook
//	    secret_env: HOOK_SECRET
//	    schedules:
//	      - {name: nightly, cron: "0 2 * * *"}
//
// and in a content type: hooks: [{on: transition, to: posted, service: ledger}]
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// maxAge is how old a signed request may be; it stops replays of captured ones.
const maxAge = 5 * time.Minute

// event is what HydroGo sends: data holds the content-type hook event.
type event struct {
	ID    string `json:"id"`
	Topic string `json:"topic"`
	Data  struct {
		Type   string         `json:"type"`   // content type id, or "schedule"
		On     string         `json:"on"`     // create, update, delete, transition, or the schedule name
		ID     string         `json:"id"`     // record id
		From   string         `json:"from"`   // transition source state
		To     string         `json:"to"`     // transition target state
		Actor  string         `json:"actor"`  // e-mail of whoever made the change
		Fields map[string]any `json:"fields"` // the submitted record
	} `json:"data"`
}

// verify checks X-Hydro-Signature = sha256=HMAC(secret, timestamp + "." + body).
func verify(secret string, r *http.Request, body []byte, now time.Time) error {
	stamp := r.Header.Get("X-Hydro-Timestamp")
	seconds, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		return errors.New("missing timestamp")
	}
	if age := now.Sub(time.Unix(seconds, 0)); age > maxAge || age < -maxAge {
		return errors.New("stale request")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, stamp+".")
	_, _ = mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(r.Header.Get("X-Hydro-Signature"))) {
		return errors.New("bad signature")
	}
	return nil
}

// handler answers hooks. 2xx lets the change through; 4xx with {"message"}
// rejects it and shows the message to the user; anything else fails closed.
func handler(secret string, now func() time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		if err := verify(secret, r, body, now()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var e event
		if err := json.Unmarshal(body, &e); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if message := decide(e); message != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// decide holds your rules. Return a message to reject the change, "" to allow
// it. Replace the example rules with your own.
func decide(e event) string {
	switch {
	case e.Data.Type == "schedule":
		log.Printf("schedule %s fired at %v", e.Data.On, e.Data.Fields["at"])
	case e.Data.On == "transition" && e.Data.To == "posted":
		// Example: refuse to post a document without a positive total.
		if total, _ := e.Data.Fields["total"].(float64); total <= 0 {
			return "A posted document needs a total above zero."
		}
	}
	return ""
}

func main() {
	secret := os.Getenv("HOOK_SECRET")
	if len(secret) < 16 {
		log.Fatal("set HOOK_SECRET to the same value HydroGo reads from secret_env (at least 16 characters)")
	}
	mux := http.NewServeMux()
	mux.Handle("POST /hook", handler(secret, time.Now))
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	fmt.Println("side-service listening on", addr)
	log.Fatal(server.ListenAndServe())
}
