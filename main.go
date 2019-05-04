package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

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
	APIAccessToken string // e.g. O8I9svDTizOfLfdVA5ri
	db             *sql.DB
	Code           env.EnvCode
}

// We don't fine grain the types like sqlNullstring since that makes the JSON
// marshaller more complex
type unit struct {
	MefeUnitID             string `json:"mefe_unit_id"`
	MefeUnitIDint          int    `json:"mefeUnitIdIntValue"`
	MefeCreatorUserID      string `json:"mefe_creator_user_id,omitempty"`
	BzfeCreatorUserID      int    `json:"bzfe_creator_user_id,omitempty"`
	ClassificationID       int    `json:"classification_id,omitempty"`
	UnitName               string `json:"unit_name,omitempty"`
	UnitDescriptionDetails string `json:"unit_description_details,omitempty"`
}

type unitCreated struct {
	ProductID int    `json:"id"`
	UnitName  string `json:"name"`
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
		`INSERT INTO ut_data_to_create_units (mefe_unit_id,
			mefe_unit_id_int_value,
			mefe_creator_user_id,
			bzfe_creator_user_id,
			classification_id,
			unit_name,
			unit_description_details
		) VALUES (?,?,?,?,?,?,?)`,
		unit.MefeUnitID,
		unit.MefeUnitIDint,
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
		DSN: fmt.Sprintf("%s:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL",
			e.GetSecret("MYSQL_USER"),
			e.GetSecret("MYSQL_PASSWORD"),
			mysqlhost),
		APIAccessToken: e.GetSecret("API_ACCESS_TOKEN"),
		Code:           e.Code,
	}

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
	app.HandleFunc("/disable", h.disableUnit).Methods("POST")
	app.HandleFunc("/", h.ping).Methods("GET")
	return app
}

func (h handler) runsql(sqlfile string, unitID string) (res sql.Result, err error) {
	if unitID == "" {
		return res, fmt.Errorf("id is unset")
	}

	sqlscript, err := ioutil.ReadFile(fmt.Sprintf("sql/%s", sqlfile))
	if err != nil {
		return
	}

	log.Infof("Running %s with unit id %v with env %d", sqlfile, unitID, h.Code)
	res, err = h.db.Exec(fmt.Sprintf(string(sqlscript), unitID, h.Code))
	return res, err
}

func (h handler) runsqlUnit(sqlfile string, u unit) (res sql.Result, err error) {
	if u.MefeUnitID == "" {
		return res, fmt.Errorf("mefe_unit_id is unset")
	}
	if u.MefeUnitIDint == 0 {
		return res, fmt.Errorf("mefeUnitIdIntValue is unset")
	}
	sqlscript, err := ioutil.ReadFile(fmt.Sprintf("sql/%s", sqlfile))
	if err != nil {
		return
	}
	log.WithFields(log.Fields{
		"unit": u,
		"env":  h.Code,
		"sql":  sqlfile,
	}).Info("exec")

	res, err = h.db.Exec(fmt.Sprintf(string(sqlscript), u.MefeUnitID, u.MefeUnitIDint, h.Code))
	return res, err
}

func (h handler) ping(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	if err != nil {
		log.WithError(err).Error("failed to ping database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "OK")
}

func (h handler) disableUnit(w http.ResponseWriter, r *http.Request) {

	// db.unitMetaData.find()
	// { "_id" : "XSr3WRWRn5QipGvjq", "bzId" : 2,
	// "bzName" : "Demo - Unit 13-06 - Comp B-2", "displayName" : "",
	// "streetAddress" : "", "city" : "", "zipCode" : "", "state" : "", "country" :
	// "", "createdAt" : ISODate("2018-06-28T02:42:18.602Z"), "ownerIds" : [ ],
	// "moreInfo" : "", "unitType" : null }

	type unitMetaData struct {
		BzID int `json:"bzId"`
	}

	decoder := json.NewDecoder(r.Body)
	var metas []unitMetaData
	err := decoder.Decode(&metas)
	if err != nil {
		log.WithError(err).Error("input error")
		response.BadRequest(w, "Invalid JSON")
		return
	}
	defer r.Body.Close()

	if len(metas) < 1 {
		response.BadRequest(w, "Empty payload")
		return
	}

	for _, umd := range metas {

		ctx := log.WithFields(log.Fields{
			"unitMetaData id": umd.BzID,
			"reqid":           r.Header.Get("X-Request-Id"),
			"UA":              r.Header.Get("User-Agent"),
		})

		_, err := h.runsql("unit_disable_existing.sql", fmt.Sprintf("%d", umd.BzID))
		if err != nil {
			ctx.WithError(err).Errorf("unit_disable_existing.sql failed")
			response.BadRequest(w, err.Error())
			return
		}
		ctx.Info("ran unit_disable_existing.sql")
	}
	response.OK(w, metas)
}

func (h handler) createUnit(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var units []unit
	err := decoder.Decode(&units)
	if err != nil {
		log.WithError(err).Error("input error")
		response.BadRequest(w, "Invalid JSON")
		return
	}
	defer r.Body.Close()

	if len(units) < 1 {
		response.BadRequest(w, "Empty payload")
		return
	}

	var results []unitCreated
	for _, unit := range units {

		ctx := log.WithFields(log.Fields{
			"unit":  unit,
			"reqid": r.Header.Get("X-Request-Id"),
			"UA":    r.Header.Get("User-Agent"),
		})

		if unit.MefeUnitID == "" {
			ctx.Error("missing ID")
			response.BadRequest(w, "Missing ID")
			return
		}

		err := h.step1Insert(unit)
		if err != nil {
			ctx.WithError(err).Error("failed to run step1Insert")
			response.BadRequest(w, err.Error())
			return
		}

		ctx.Info("inserted")

		start := time.Now()
		_, err = h.runsqlUnit("unit_create_new.sql", unit)
		if err != nil {
			ctx.WithError(err).Errorf("unit_create_new.sql failed")
			response.BadRequest(w, err.Error())
			return
		}
		ctx.WithField("duration", time.Since(start)).Infof("ran unit_create_new.sql")
		ProductID, err := h.getProductID(unit.MefeUnitIDint)
		if err != nil {
			ctx.WithError(err).Errorf("unit_create_new.sql failed")
			response.BadRequest(w, err.Error())
			return
		}
		results = append(results, ProductID)
	}

	response.OK(w, results)

}

func (h handler) getProductID(MefeUnitIDint int) (newUnit unitCreated, err error) {
	err = h.db.QueryRow("SELECT product_id FROM ut_data_to_create_units WHERE mefe_unit_id_int_value=?", MefeUnitIDint).
		Scan(&newUnit.ProductID)
	if err != nil {
		return newUnit, err
	}
	err = h.db.QueryRow("SELECT name FROM products WHERE id=?", newUnit.ProductID).
		Scan(&newUnit.UnitName)
	return newUnit, err
}
