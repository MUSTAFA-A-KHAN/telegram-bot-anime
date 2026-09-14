package en

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/dictionary"
)

var _ dictionary.Dictionary = (*Wiktionary)(nil)

// redirectTransport rewrites every request to hit the test server, preserving the path.
type redirectTransport struct {
	target *url.URL
}

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.URL.Scheme = t.target.Scheme
	r.URL.Host = t.target.Host
	r.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

type errTransport struct{}

func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network down")
}

func newTestWiktionary(t *testing.T, handler http.HandlerFunc) *Wiktionary {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	return &Wiktionary{client: &http.Client{Transport: redirectTransport{target: target}}}
}

func TestGetMeaning_Success(t *testing.T) {
	var gotPath string
	w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{
			"en": [
				{"partOfSpeech": "Noun", "definitions": [
					{"definition": "A domesticated feline."},
					{"definition": ""},
					{"definition": "A spiteful woman."}
				]},
				{"partOfSpeech": "Verb", "definitions": [
					{"definition": "To hoist the anchor."}
				]}
			],
			"fr": [
				{"partOfSpeech": "Noun", "definitions": [{"definition": "should be ignored"}]}
			]
		}`))
	})

	got, err := w.GetMeaning("  cat  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Noun:\n1. A domesticated feline.\n2. A spiteful woman.\nVerb:\n1. To hoist the anchor."
	if got != want {
		t.Errorf("GetMeaning() = %q, want %q", got, want)
	}
	if gotPath != "/api/rest_v1/page/definition/cat" {
		t.Errorf("request path = %q, want trimmed word in path", gotPath)
	}
}

func TestGetMeaning_EscapesWord(t *testing.T) {
	var gotRawPath string
	w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
		gotRawPath = r.URL.EscapedPath()
		rw.Write([]byte(`{"en":[{"partOfSpeech":"Noun","definitions":[{"definition":"x"}]}]}`))
	})

	if _, err := w.GetMeaning("ice cream/cone"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(gotRawPath, "/ice%20cream%2Fcone") {
		t.Errorf("escaped path = %q, want word to be path-escaped", gotRawPath)
	}
}

func TestGetMeaning_DefinitionsWithoutPartOfSpeech(t *testing.T) {
	w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(`{"en":[{"definitions":[{"definition":"only a definition"}]}]}`))
	})

	got, err := w.GetMeaning("word")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1. only a definition" {
		t.Errorf("GetMeaning() = %q", got)
	}
}

func TestCleanDefinition(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain text", in: "A fruit.", want: "A fruit."},
		{
			name: "wiki links and spans",
			in:   `A <a rel="mw:WikiLink" href="/wiki/round" title="round">round</a> <span class="biota"><i>Malus</i></span>.`,
			want: "A round Malus.",
		},
		{
			name: "html entities",
			in:   `noun sense<span typeof="mw:Entity">&nbsp;</span>1.1 &amp; more &quot;x&quot;`,
			want: `noun sense 1.1 & more "x"`,
		},
		{
			name: "style blocks removed",
			in:   `Eating. <style data-mw-deduplicate="x" typeof="mw:Extension/templatestyles">.mw-parser-output .defdate{font-size:smaller}</style>`,
			want: "Eating.",
		},
		{
			name: "self-closing link tag removed",
			in:   `Forbidden fruit. <link rel="mw-deduplicated-inline-style" href="mw-data:TemplateStyles:r1">`,
			want: "Forbidden fruit.",
		},
		{
			name: "nested sub-sense list dropped",
			in:   "Something round.\n\n<ol><li>Ellipsis of Adam's apple.</li>\n<li>Ellipsis of apple-green.",
			want: "Something round.",
		},
		{
			name: "only a nested list",
			in:   `<span class="usage-label-sense"></span><ol><li>A person.</li></ol>`,
			want: "",
		},
		{name: "backticks replaced", in: "use `code` here", want: "use 'code' here"},
		{name: "whitespace collapsed", in: "  a \n\t b  ", want: "a b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanDefinition(tt.in); got != tt.want {
				t.Errorf("cleanDefinition() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMeanings_DedupesAndLimitsPerPOS(t *testing.T) {
	entries := []Entry{
		{PartOfSpeech: "Noun", Definitions: []Definition{
			{Definition: "<b>One</b>."},
			{Definition: "One."}, // duplicate after cleaning
			{Definition: `<ol><li>nested</li></ol>`},
			{Definition: "Two."},
			{Definition: "Three."},
			{Definition: "Four."}, // over the per-POS limit
		}},
		{PartOfSpeech: "Adjective", Definitions: []Definition{{Definition: "<i></i>"}}}, // nothing left, section skipped
		{PartOfSpeech: "Verb", Definitions: []Definition{{Definition: "One."}, {Definition: "Act."}}},
	}

	got := formatMeanings(entries)
	want := "Noun:\n1. One.\n2. Two.\n3. Three.\nVerb:\n1. Act."
	if got != want {
		t.Errorf("formatMeanings() = %q, want %q", got, want)
	}
}

func TestFormatMeanings_RespectsMaxLength(t *testing.T) {
	long := strings.Repeat("word ", 50) // ~250 runes per definition
	var entries []Entry
	for _, pos := range []string{"Noun", "Verb", "Adjective", "Adverb"} {
		entries = append(entries, Entry{PartOfSpeech: pos, Definitions: []Definition{
			{Definition: pos + " " + long},
			{Definition: pos + " again " + long},
		}})
	}

	got := formatMeanings(entries)
	if n := utf8.RuneCountInString(got); n > maxMeaningLength {
		t.Fatalf("length = %d runes, want <= %d", n, maxMeaningLength)
	}
	if !strings.HasPrefix(got, "Noun:\n1. ") {
		t.Errorf("expected first section to be kept, got %q", got)
	}
	if strings.Contains(got, "Adverb:") {
		t.Errorf("expected later sections to be dropped once limit reached, got %q", got)
	}
}

func TestFormatMeanings_TruncatesSingleHugeDefinition(t *testing.T) {
	entries := []Entry{{PartOfSpeech: "Noun", Definitions: []Definition{
		{Definition: strings.Repeat("é", maxMeaningLength*2)},
	}}}

	got := formatMeanings(entries)
	if n := utf8.RuneCountInString(got); n != maxMeaningLength {
		t.Errorf("length = %d runes, want %d", n, maxMeaningLength)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix, got %q", got)
	}
}

func TestGetMeaning_Errors(t *testing.T) {
	tests := []struct {
		name    string
		word    string
		body    string
		wantErr string
	}{
		{name: "empty word", word: "", wantErr: "word cannot be empty"},
		{name: "whitespace word", word: "   \t", wantErr: "word cannot be empty"},
		{name: "invalid json", word: "cat", body: `not json`, wantErr: "decode wiktionary response"},
		{name: "no english entry", word: "chat", body: `{"fr":[{"partOfSpeech":"Noun","definitions":[{"definition":"cat"}]}]}`, wantErr: `no English definition found for "chat"`},
		{name: "empty english entries", word: "cat", body: `{"en":[]}`, wantErr: `no definition found for "cat"`},
		{name: "blank definitions", word: "cat", body: `{"en":[{"definitions":[{"definition":""}]}]}`, wantErr: `no definition found for "cat"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
				called = true
				rw.Write([]byte(tt.body))
			})

			got, err := w.GetMeaning(tt.word)
			if err == nil {
				t.Fatalf("expected error containing %q, got result %q", tt.wantErr, got)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
			if got != "" {
				t.Errorf("result = %q, want empty on error", got)
			}
			if strings.TrimSpace(tt.word) == "" && called {
				t.Error("expected no HTTP request for empty word")
			}
		})
	}
}

func TestGetMeaning_StatusErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{name: "not found", status: http.StatusNotFound, wantErr: `word "cat" not found`},
		{name: "forbidden", status: http.StatusForbidden, wantErr: "wiktionary returned status 403"},
		{name: "server error", status: http.StatusInternalServerError, wantErr: "wiktionary returned status 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
				http.Error(rw, "Please set a user-agent", tt.status)
			})

			_, err := w.GetMeaning("cat")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetMeaning_SetsUserAgent(t *testing.T) {
	var gotUA string
	w := newTestWiktionary(t, func(rw http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		rw.Write([]byte(`{"en":[{"definitions":[{"definition":"x"}]}]}`))
	})

	if _, err := w.GetMeaning("cat"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, userAgent)
	}
}

func TestGetMeaning_TransportError(t *testing.T) {
	w := &Wiktionary{client: &http.Client{Transport: errTransport{}}}

	_, err := w.GetMeaning("cat")
	if err == nil || !strings.Contains(err.Error(), "request wiktionary") {
		t.Fatalf("error = %v, want request wiktionary error", err)
	}
}

// TestGetMeaning_RealAPI calls the live Wiktionary API. Skip with `go test -short`.
func TestGetMeaning_RealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Wiktionary API test in short mode")
	}

	w := NewWiktionary()

	t.Run("known word", func(t *testing.T) {
		got, err := w.GetMeaning("apple")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "Noun:") {
			t.Errorf("expected a Noun section, got %q", got)
		}
		if strings.ContainsAny(got, "<>`") {
			t.Errorf("expected plain text without HTML or backticks, got %q", got)
		}
		if n := utf8.RuneCountInString(got); n > maxMeaningLength {
			t.Errorf("length = %d runes, want <= %d", n, maxMeaningLength)
		}
		t.Logf("apple:\n%s", got)
	})

	t.Run("nonexistent word", func(t *testing.T) {
		got, err := w.GetMeaning("qwzxvbnmplk")
		if err == nil {
			t.Fatalf("expected error for nonexistent word, got %q", got)
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want a not found error", err)
		}
	})
}

func TestNewWiktionary(t *testing.T) {
	w := NewWiktionary()
	if w == nil || w.client == nil {
		t.Fatal("NewWiktionary() returned nil or missing client")
	}
}
