package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var h handler

func TestMain(m *testing.M) {
	h, _ = New()
	defer h.db.Close()
	os.Exit(m.Run())
}

func TestRecorder(t *testing.T) {
	type checkFunc func(*httptest.ResponseRecorder) error
	check := func(fns ...checkFunc) []checkFunc { return fns }

	hasStatus := func(want int) checkFunc {
		return func(rec *httptest.ResponseRecorder) error {
			if rec.Code != want {
				return fmt.Errorf("expected status %d, found %d", want, rec.Code)
			}
			return nil
		}
	}
	containsContents := func(want string) checkFunc {
		return func(rec *httptest.ResponseRecorder) error {
			if have := rec.Body.String(); !strings.Contains(have, want) {
				return fmt.Errorf("expected to find %q, in %q", want, have)
			}
			return nil
		}
	}
	hasHeader := func(key, want string) checkFunc {
		return func(rec *httptest.ResponseRecorder) error {
			if have := rec.Result().Header.Get(key); have != want {
				return fmt.Errorf("expected header %s: %q, found %q", key, want, have)
			}
			return nil
		}
	}

	tests := [...]struct {
		name    string
		verb    string
		path    string
		payload io.Reader
		h       func(w http.ResponseWriter, r *http.Request)
		checks  []checkFunc
	}{
		{
			"Empty payload",
			"POST",
			"/create",
			nil,
			h.createUnit,
			check(hasStatus(400)),
		},
		{
			"Ping",
			"GET",
			"/",
			nil,
			h.ping,
			check(hasStatus(200), containsContents("OK"), hasHeader("Content-Type", "text/plain; charset=utf-8")),
		},
	}

	for _, tt := range tests {
		r, err := http.NewRequest(tt.verb, tt.path, tt.payload)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(tt.h)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)
			for _, check := range tt.checks {
				if err := check(rec); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
