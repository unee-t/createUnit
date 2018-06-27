package main

import (
	"net/http"
	"os"
	"testing"

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
	r.GET("/").
		Run(h.BasicEngine(), func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
}
