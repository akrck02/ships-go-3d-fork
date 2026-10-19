// Package i18n translates the messages the server sends to players. The site
// chooses a language and sends it as Accept-Language (or, on the game
// websocket, as a "lang" query parameter, since a browser can't set headers
// there); anything unsupported gets English.
package i18n

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Lang string

const (
	En Lang = "en"
	Es Lang = "es"
	Eu Lang = "eu"
)

var supported = []Lang{En, Es, Eu}

// Parse picks the first supported language in an Accept-Language style list
// ("eu-ES,eu;q=0.9,es;q=0.8"). Weights are ignored: browsers list languages
// in order of preference anyway.
func Parse(value string) Lang {
	for _, item := range strings.Split(value, ",") {
		tag, _, _ := strings.Cut(item, ";")
		tag, _, _ = strings.Cut(strings.TrimSpace(tag), "-")
		for _, lang := range supported {
			if strings.EqualFold(tag, string(lang)) {
				return lang
			}
		}
	}
	return En
}

// FromRequest is the language a request asks for: its "lang" query
// parameter, else its Accept-Language header.
func FromRequest(r *http.Request) Lang {
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return Parse(lang)
	}
	return Parse(r.Header.Get("Accept-Language"))
}

// Message is a translatable message: a key into the catalog and the
// arguments for its format string. Arguments that are Messages themselves
// are translated too. As an error, it reads in English.
type Message struct {
	Key  string
	Args []any
}

func New(key string, args ...any) Message {
	return Message{Key: key, Args: args}
}

func (m Message) Error() string { return m.In(En) }

func (m Message) In(lang Lang) string {
	formats, ok := catalog[m.Key]
	if !ok {
		return m.Key
	}
	format, ok := formats[lang]
	if !ok {
		format = formats[En]
	}
	args := make([]any, len(m.Args))
	for i, arg := range m.Args {
		if inner, ok := arg.(Message); ok {
			arg = inner.In(lang)
		}
		args[i] = arg
	}
	return fmt.Sprintf(format, args...)
}

// Translate returns err in lang when it is (or wraps) a Message, and its
// plain text otherwise.
func Translate(err error, lang Lang) string {
	var m Message
	if errors.As(err, &m) {
		return m.In(lang)
	}
	return err.Error()
}
