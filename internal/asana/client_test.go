package asana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// noBackoff overrides the package-level backoff var for the duration of a
// test, restoring the original on cleanup, so retry tests don't actually
// sleep for real seconds.
func noBackoff(t *testing.T) {
	t.Helper()
	orig := backoff
	backoff = func() time.Duration { return time.Millisecond }
	t.Cleanup(func() { backoff = orig })
}

func newTestClient(t *testing.T, serverURL string, maxAttempts int) *Client {
	t.Helper()
	return NewClient(http.DefaultClient, "test-token", serverURL, "workspace-1", rate.NewLimiter(rate.Inf, 1), maxAttempts, 1)
}

func TestGet_SetsAuthAndWorkspaceQueryParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		if got := r.URL.Query().Get("workspace"); got != "workspace-1" {
			t.Errorf("workspace query param = %q, want %q", got, "workspace-1")
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit query param = %q, want %q", got, "5")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 1)
	status, _, body, err := client.get(context.Background(), "/users", url.Values{"limit": {"5"}})
	if err != nil {
		t.Fatalf("get returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
	if string(body) != `{"data":[]}` {
		t.Errorf("body = %q", string(body))
	}
}

func TestGetWithRetries_RetriesOn5xxThenSucceeds(t *testing.T) {
	noBackoff(t)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 3)
	body, err := client.getWithRetries(context.Background(), "/users", url.Values{})
	if err != nil {
		t.Fatalf("getWithRetries returned error: %v", err)
	}
	if string(body) != `{"data":[]}` {
		t.Errorf("body = %q", string(body))
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server received %d call(s), want 2", got)
	}
}

func TestGetWithRetries_RetriesOn429UsingRetryAfterHeader(t *testing.T) {
	noBackoff(t)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"errors":[{"message":"rate limited"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 3)
	body, err := client.getWithRetries(context.Background(), "/users", url.Values{})
	if err != nil {
		t.Fatalf("getWithRetries returned error: %v", err)
	}
	if string(body) != `{"data":[]}` {
		t.Errorf("body = %q", string(body))
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server received %d call(s), want 2", got)
	}
}

func TestGetWithRetries_ClientErrorReturnsImmediately(t *testing.T) {
	noBackoff(t)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"message":"bad request"}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 3)
	if _, err := client.getWithRetries(context.Background(), "/users", url.Values{}); err == nil {
		t.Fatal("expected an error for a 400 response, got nil")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server received %d call(s), want 1 (no retry on 4xx)", got)
	}
}

func TestGetWithRetries_ExhaustsMaxAttempts(t *testing.T) {
	noBackoff(t)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 2)
	if _, err := client.getWithRetries(context.Background(), "/users", url.Values{}); err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server received %d call(s), want 2 (== maxAttempts)", got)
	}
}

func TestGetAllPages_FollowsNextPageUntilEmpty(t *testing.T) {
	type item struct {
		GID string `json:"gid"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset := r.URL.Query().Get("offset")
		w.WriteHeader(http.StatusOK)
		switch offset {
		case "":
			json.NewEncoder(w).Encode(paginatedResponse[item]{
				Data:     []item{{GID: "1"}},
				NextPage: &NextPage{Offset: "page2"},
			})
		case "page2":
			json.NewEncoder(w).Encode(paginatedResponse[item]{
				Data:     []item{{GID: "2"}},
				NextPage: nil,
			})
		default:
			t.Errorf("unexpected offset %q", offset)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 1)
	items, err := GetAllPages[item](context.Background(), client, "/items", Pagination{Limit: 1})
	if err != nil {
		t.Fatalf("GetAllPages returned error: %v", err)
	}

	want := []item{{GID: "1"}, {GID: "2"}}
	if len(items) != len(want) || items[0] != want[0] || items[1] != want[1] {
		t.Errorf("items = %+v, want %+v", items, want)
	}
}

func TestExtractUsers_PaginatesAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		offset := r.URL.Query().Get("offset")
		w.WriteHeader(http.StatusOK)
		switch offset {
		case "":
			json.NewEncoder(w).Encode(paginatedResponse[User]{
				Data:     []User{{GID: "1", Name: "Alice"}},
				NextPage: &NextPage{Offset: "next"},
			})
		case "next":
			json.NewEncoder(w).Encode(paginatedResponse[User]{
				Data: []User{{GID: "2", Name: "Bob"}},
			})
		default:
			t.Fatalf("unexpected offset %q", offset)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 1)
	users, err := client.ExtractUsers(context.Background())
	if err != nil {
		t.Fatalf("ExtractUsers returned error: %v", err)
	}
	if len(users) != 2 || users[0].GID != "1" || users[1].GID != "2" {
		t.Errorf("users = %+v", users)
	}
}

func TestExtractProjects_PaginatesAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		offset := r.URL.Query().Get("offset")
		w.WriteHeader(http.StatusOK)
		switch offset {
		case "":
			json.NewEncoder(w).Encode(paginatedResponse[Project]{
				Data:     []Project{{GID: "10", Name: "Website"}},
				NextPage: &NextPage{Offset: "next"},
			})
		case "next":
			json.NewEncoder(w).Encode(paginatedResponse[Project]{
				Data: []Project{{GID: "20", Name: "Mobile"}},
			})
		default:
			t.Fatalf("unexpected offset %q", offset)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL, 1)
	projects, err := client.ExtractProjects(context.Background())
	if err != nil {
		t.Fatalf("ExtractProjects returned error: %v", err)
	}
	if len(projects) != 2 || projects[0].GID != "10" || projects[1].GID != "20" {
		t.Errorf("projects = %+v", projects)
	}
}

func TestPagination_QueryParams(t *testing.T) {
	tests := []struct {
		name string
		p    Pagination
		want url.Values
	}{
		{"empty", Pagination{}, url.Values{}},
		{"limit only", Pagination{Limit: 10}, url.Values{"limit": {"10"}}},
		{"offset only", Pagination{Offset: "abc"}, url.Values{"offset": {"abc"}}},
		{"both", Pagination{Limit: 10, Offset: "abc"}, url.Values{"limit": {"10"}, "offset": {"abc"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.QueryParams()
			if got.Encode() != tt.want.Encode() {
				t.Errorf("QueryParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
