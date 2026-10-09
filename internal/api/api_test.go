package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"backend/internal/api"
	"backend/internal/store"
)

// newServer starts the real API against a real Postgres. Each test gets its
// own empty schema, so tests never see each other's rows or your local data.
func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL not set; see README, section Tests")
	}

	admin, err := sql.Open("postgres", base)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	ctx := context.Background()
	st, err := store.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(api.New(st, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return srv
}

// call sends a request (with body as JSON, if not nil), decodes the JSON reply
// into out (if not nil), and returns the HTTP status.
func call(t *testing.T, srv *httptest.Server, method, path string, body, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: reply is not JSON: %v", method, path, err)
		}
	}
	return res.StatusCode
}

func TestHealth(t *testing.T) {
	srv := newServer(t)

	var reply map[string]string
	status := call(t, srv, "GET", "/healthz", nil, &reply)

	if status != http.StatusOK || reply["status"] != "ok" {
		t.Fatalf("status %d, reply %v", status, reply)
	}
}

// TODO: your tests. Copy TestHealth: start a server, call an endpoint, check the reply.
