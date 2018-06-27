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
	"github.com/gorilla/mux"
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

type unit struct {
	MefeUnitID             sql.NullString `json:"mefe_unit_id"`             // mefe_unit_id
	MefeCreatorUserID      sql.NullString `json:"mefe_creator_user_id"`     // mefe_creator_user_id
	BzfeCreatorUserID      int            `json:"bzfe_creator_user_id"`     // bzfe_creator_user_id
	ClassificationID       int16          `json:"classification_id"`        // classification_id
	UnitName               string         `json:"unit_name"`                // unit_name
	UnitDescriptionDetails sql.NullString `json:"unit_description_details"` // unit_description_details
}

func init() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonhandler.Default)
	}
}

func (h handler) step1Insert(unit unit) (err error) {
	_, err = h.db.Exec(
		`INSERT INTO ut_data_to_create_units (mefe_unit_id
			mefe_creator_user_id,
			bzfe_creator_user_id,
			classification_id,
			unit_name,
			unit_description_details
		) VALUES (?,?,?,?,?,?)`,
		unit.MefeUnitID,
		unit.MefeCreatorUserID,
		unit.BzfeCreatorUserID,
		unit.ClassificationID,
		unit.UnitName,
		unit.UnitDescriptionDetails,
	)
	return
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
		log.WithError(err).Warn("error getting unee-t env")
	}

	var mysqlhost string
	val, ok := os.LookupEnv("MYSQL_HOST")
	if ok {
		log.Infof("MYSQL_HOST overridden by local env: %s", val)
		mysqlhost = val
	} else {
		mysqlhost = e.Udomain("auroradb")
	}

	h = handler{
		DSN: fmt.Sprintf("bugzilla:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL",
			e.GetSecret("MYSQL_PASSWORD"),
			mysqlhost),
		Domain:         fmt.Sprintf("https://%s", e.Udomain("case")),
		APIAccessToken: e.GetSecret("API_ACCESS_TOKEN"),
		Code:           e.Code,
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
	app := h.BasicEngine()

	if err := http.ListenAndServe(addr, env.Protect(app, h.APIAccessToken)); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func (h handler) BasicEngine() http.Handler {
	app := mux.NewRouter()
	app.HandleFunc("/create", h.createUnit).Methods("POST")
	return app
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
	var units []unit
	err := decoder.Decode(&units)
	if err != nil {
		log.WithError(err).Error("input error")
		response.BadRequest(w, "Invalid JSON")
		return
	}
	defer r.Body.Close()

	for _, unit := range units {

		ctx := log.WithFields(log.Fields{
			"unit": unit,
		})

		if !unit.MefeUnitID.Valid {
			ctx.Error("missing ID")
			response.BadRequest(w, "Missing ID")
			return
		}

		err := h.step1Insert(unit)
		if err != nil {
			ctx.WithError(err).Error("failed to run step1Insert")
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		ctx.Info("inserted")

		err = h.runsql("unit_create_new.sql", unit.MefeUnitID.String)
		if err != nil {
			ctx.WithError(err).Errorf("unit_create_new.sql failed")
			response.BadRequest(w, err.Error())
			return
		}
		ctx.Info("ran unit_create_new.sql")
	}

	response.OK(w, units)

}
