package i18n

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := map[string]Lang{
		"":                        En,
		"es":                      Es,
		"eu-ES,eu;q=0.9,es;q=0.8": Eu,
		"fr-FR, es-ES;q=0.8":      Es,
		"de":                      En,
		"EU":                      Eu,
		"en-GB,en;q=0.9,eu;q=0.8": En,
	}
	for value, want := range cases {
		if got := Parse(value); got != want {
			t.Errorf("Parse(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestFromRequestPrefersQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws?lang=eu", nil)
	r.Header.Set("Accept-Language", "es")
	if got := FromRequest(r); got != Eu {
		t.Errorf("FromRequest = %q, want eu", got)
	}
	r = httptest.NewRequest("GET", "/ws", nil)
	r.Header.Set("Accept-Language", "es")
	if got := FromRequest(r); got != Es {
		t.Errorf("FromRequest = %q, want es", got)
	}
}

var (
	verbPattern = regexp.MustCompile(`%([a-z])`)
	sampleArgs  = map[string]any{"d": 1, "g": 1.5, "q": "x", "s": "x"}
)

// Every message exists in every language, with the same arguments: a
// translation that drops or adds a verb would print %!d(MISSING) or
// %!(EXTRA ...) to players.
func TestCatalogComplete(t *testing.T) {
	for key, formats := range catalog {
		var args []any
		for _, verb := range verbPattern.FindAllStringSubmatch(formats[En], -1) {
			args = append(args, sampleArgs[verb[1]])
		}
		for _, lang := range supported {
			format, ok := formats[lang]
			if !ok {
				t.Errorf("%s: missing %s", key, lang)
				continue
			}
			if out := fmt.Sprintf(format, args...); strings.Contains(out, "%!") {
				t.Errorf("%s (%s): %q", key, lang, out)
			}
		}
	}
}

func TestNestedAndWrapped(t *testing.T) {
	m := New("layerPart", "Hull", 2, New("invalidRotation"))
	if got := m.In(Es); got != `capa "Hull", pieza 2: rotación no válida` {
		t.Errorf("got %q", got)
	}
	if got := m.Error(); got != `layer "Hull", part 2: invalid rotation` {
		t.Errorf("got %q", got)
	}
	wrapped := fmt.Errorf("saving: %w", New("invalidRotation"))
	if got := Translate(wrapped, Eu); got != "biraketa baliogabea" {
		t.Errorf("got %q", got)
	}
	if got := Translate(errors.New("plain"), Eu); got != "plain" {
		t.Errorf("got %q", got)
	}
}
