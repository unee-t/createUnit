package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pallinder/go-randomdata"
)

type checkFunc func(*httptest.ResponseRecorder) error

func hasStatus(want int) checkFunc {
	return func(rec *httptest.ResponseRecorder) error {
		if rec.Code != want {
			return fmt.Errorf("expected status %d, found %d", want, rec.Code)
		}
		return nil
	}
}

func hasHeader(key, want string) checkFunc {
	return func(rec *httptest.ResponseRecorder) error {
		if have := rec.Result().Header.Get(key); have != want {
			return fmt.Errorf("expected header %s: %q, found %q", key, want, have)
		}
		return nil
	}
}

func TestConnected(t *testing.T) {
	h, _ := NewDbConnexion()
	defer h.db.Close()

	check := func(fns ...checkFunc) []checkFunc { return fns }

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

func TestBadConnection(t *testing.T) {
	h, _ := NewDbConnexion()
	// Bad because we close the connection here
	h.db.Close()

	check := func(fns ...checkFunc) []checkFunc { return fns }

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
			check(hasStatus(500)),
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
			check(hasStatus(500)),
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
