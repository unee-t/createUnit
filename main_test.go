package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/Pallinder/go-randomdata"
	"github.com/appleboy/gofight"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

var h handler

func TestMain(m *testing.M) {
	h, _ = New()
	defer h.db.Close()
	os.Exit(m.Run())
}

func TestRoutes(t *testing.T) {
	r := gofight.New()
	r.POST("/create").
		SetBody("[]").
		Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, "Empty payload\n", r.Body.String())
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
	r.POST("/create").
		SetBody("notJSON").
		Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, "Invalid JSON\n", r.Body.String())
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
	r.POST("/disable").
		SetBody("").
		Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, "Invalid JSON\n", r.Body.String())
			assert.Equal(t, http.StatusBadRequest, r.Code)
		})
	r.GET("/").
		Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
		})
}

func Benchmark_RealCreate(b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		name := randomdata.SillyName()
		r := gofight.New()
		u := []unit{unit{
			MefeUnitID:             name,
			MefeCreatorUserID:      "user",
			BzfeCreatorUserID:      55,
			ClassificationID:       2,
			UnitName:               name,
			UnitDescriptionDetails: "Up on the hills and testing",
		}}
		uJSON, _ := json.Marshal(u)
		r.POST("/create").
			SetBody(string(uJSON)).
			Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
				assert.Contains(b, r.Body.String(), name)
				assert.Equal(b, http.StatusOK, r.Code)
			})
	}
}
