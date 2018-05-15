package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/gorilla/pat"
	"github.com/tj/go/http/response"
	"github.com/unee-t/env"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type handler struct {
	DSN            string // e.g. "bugzilla:secret@tcp(auroradb.dev.unee-t.com:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL"
	Domain         string // e.g. https://dev.case.unee-t.com
	APIAccessToken string // e.g. O8I9svDTizOfLfdVA5ri
	db             *sql.DB
	Code           env.EnvCode
}

type Unit struct {
	ID string `json:"id"`
}

func init() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonhandler.Default)
	}
}

// New setups the configuration assuming various parameters have been setup in the AWS account
func New() (h handler, err error) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("uneet-dev"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID
	e, err := env.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("error getting unee-t env")
	}

	h = handler{
		DSN: fmt.Sprintf("bugzilla:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL",
			e.GetSecret("MYSQL_PASSWORD"),
			e.Udomain("auroradb")),
		Domain:         fmt.Sprintf("https://%s", e.Udomain("case")),
		APIAccessToken: e.GetSecret("API_ACCESS_TOKEN"),
		Code:           e.Code,
	}

	if h.Code == 0 {
		err = fmt.Errorf("Error code is unknown/unset")
		return
	}

	log.Infof("Frontend URL: %v", h.Domain)

	h.db, err = sql.Open("mysql", h.DSN)
	if err != nil {
		log.WithError(err).Fatal("error opening database")
		return
	}

	return

}

func main() {

	h, err := New()
	if err != nil {
		log.WithError(err).Fatal("error setting configuration")
		return
	}

	defer h.db.Close()

	addr := ":" + os.Getenv("PORT")
	app := pat.New()
	app.Post("/create", http.HandlerFunc(h.createUnit))
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func (h handler) runsql(sqlfile string, unitID string) (err error) {
	sqlscript, err := ioutil.ReadFile(fmt.Sprintf("sql/%s", sqlfile))
	if err != nil {
		return
	}
	log.Infof("Running %s with unit id %s with env %d", sqlfile, unitID, h.Code)
	_, err = h.db.Exec(fmt.Sprintf(string(sqlscript), unitID, h.Code))
	return
}

func (h handler) createUnit(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("X-Robots-Tag", "none") // We don't want Google to index us

	decoder := json.NewDecoder(r.Body)
	var u Unit
	err := decoder.Decode(&u)
	if err != nil {
		log.WithError(err).Errorf("Input error")
		response.BadRequest(w, "Invalid JSON")
		return
	}
	defer r.Body.Close()

	ctx := log.WithFields(log.Fields{
		"unit": u,
	})

	ctx.Info("load json payload")

	if u.ID == "" {
		ctx.Error("Missing ID")
		response.BadRequest(w, "Missing ID")
		return
	}

	err = h.runsql("unit_create_new.sql", u.ID)
	if err != nil {
		ctx.WithError(err).Errorf("unit_create_new.sql failed")
		response.BadRequest(w, "Failed to create new unit")
		return
	}

	response.OK(w, u)

}
