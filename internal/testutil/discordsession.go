package testutil

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
)

/*
Handler tests need a *discordgo.Session that doesn't hit the network.

A plain `discordgo.New(...)` session is fully initialized (ratelimiter, state, http
client). Therefore, swapping its Client.Transport is enough to stop real REST calls
while letting the handler code path run normally.

Captured requests are available via Requests() for assertions that need to know
what the handler told Discord.
*/

// RecordedRequest captures a single REST call intercepted by the stub transport.
type RecordedRequest struct {
	Method string
	URL    string
	Body   []byte
}

/*
StubSession wraps a real *discordgo.Session whose HTTP client is redirected
into an in-memory recorder.

Note:

	REST calls always succeed with 200 OK + `{}`.
*/
type StubSession struct {
	*discordgo.Session
	transport *stubTransport
}

// NewStubSession builds a Session that captures REST calls without hitting Discord.
func NewStubSession(t *testing.T) *StubSession {
	t.Helper()

	s, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("stub session: %v", err)
	}

	tr := &stubTransport{}
	s.Client = &http.Client{Transport: tr}
	return &StubSession{Session: s, transport: tr}
}

// Requests returns a snapshot of every REST call the stub intercepted, in order.
func (s *StubSession) Requests() []RecordedRequest {
	return s.transport.snapshot()
}

type stubTransport struct {
	mu       sync.Mutex
	captured []RecordedRequest
}

/*
userLookupPath matches /users/<id> where <id> is not "@me" and has no further
path segments.

discordgo's UserValue(s) calls s.User(id) which hits this path;
returning 404 here makes UserValue fall back to &User{ID: id} from its option
data, preserving the target ID in command tests.
*/
var userLookupPath = regexp.MustCompile(`/users/[^/@][^/]*$`)

func (t *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
		_ = r.Body.Close()
	}

	t.mu.Lock()
	t.captured = append(t.captured, RecordedRequest{
		Method: r.Method,
		URL:    r.URL.String(),
		Body:   body,
	})
	t.mu.Unlock()

	status := http.StatusOK
	if r.Method == http.MethodGet && userLookupPath.MatchString(r.URL.Path) {
		status = http.StatusNotFound
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("{}")),
		Header:     make(http.Header),
		Request:    r,
	}, nil
}

func (t *stubTransport) snapshot() []RecordedRequest {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]RecordedRequest, len(t.captured))
	copy(out, t.captured)
	return out
}
