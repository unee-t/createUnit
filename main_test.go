package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Pallinder/go-randomdata"
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

	payload := func(u []unit) io.Reader {
		requestByte, _ := json.Marshal(u)
		return bytes.NewReader(requestByte)
	}

	unitName := randomdata.SillyName()

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
		{
			"New",
			"POST",
			"/create",
			payload([]unit{unit{
				MefeUnitID:             unitName,
				MefeCreatorUserID:      "user",
				BzfeCreatorUserID:      55,
				ClassificationID:       2,
				UnitName:               unitName,
				UnitDescriptionDetails: "Up on the hills and testing",
			}}),
			h.createUnit,
			check(hasStatus(200), hasHeader("Content-Type", "application/json")),
		},
		{
			"Replay",
			"POST",
			"/create",
			payload([]unit{unit{
				MefeUnitID:             unitName,
				MefeCreatorUserID:      "user",
				BzfeCreatorUserID:      55,
				ClassificationID:       2,
				UnitName:               unitName,
				UnitDescriptionDetails: "Up on the hills and testing",
			}}),
			h.createUnit,
			check(hasStatus(200), hasHeader("Content-Type", "application/json")),
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
