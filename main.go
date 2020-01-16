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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tj/go/http/response"
	"github.com/unee-t-ins/env"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var pingPollingFreq = 5 * time.Second

type handler struct {
	DSN            string // aurora database connection string
	APIAccessToken string
	db             *sql.DB
	Code           env.EnvCode
}

// We don't fine grain the types like sqlNullstring since that makes the JSON
// marshaller more complex
type unit struct {
	MefeUnitID             string `json:"mefe_unit_id"`
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

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("ins-dev"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID
	e, err := env.New(cfg)
	if err != nil {
		log.WithError(err).Warn("error getting unee-t env")
	}

	h = handler{
		DSN:            e.BugzillaDSN(),
		APIAccessToken: e.GetSecret("API_ACCESS_TOKEN"),
		Code:           e.Code,
	}

	h.db, err = sql.Open("mysql", h.DSN)
	if err != nil {
		log.WithError(err).Fatal("error opening database")
		return
	}

	microservicecheck := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "microservice",
			Help: "Version with DB ping check",
		},
		[]string{
			"commit",
		},
	)

	version := os.Getenv("UP_COMMIT")

	go func() {
		for {
			if h.db.Ping() == nil {
				microservicecheck.WithLabelValues(version).Set(1)
			} else {
				microservicecheck.WithLabelValues(version).Set(0)
			}
			time.Sleep(pingPollingFreq)
		}
	}()

	err = prometheus.Register(microservicecheck)
	if err != nil {
		log.Warn("prom already registered")
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
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}
}

func (h handler) BasicEngine() http.Handler {
	app := mux.NewRouter()
	app.HandleFunc("/create", h.createUnit).Methods("POST")
	app.HandleFunc("/disable", h.disableUnit).Methods("POST")
	app.Handle("/metrics", promhttp.Handler()).Methods("GET")

	if os.Getenv("UP_STAGE") == "" {
		// local dev, get around permissions
		return app
	}
	return env.Protect(app, h.APIAccessToken)
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
	sqlscript, err := ioutil.ReadFile(fmt.Sprintf("sql/%s", sqlfile))
	if err != nil {
		return
	}
	log.WithFields(log.Fields{
		"unit": u,
		"env":  h.Code,
		"sql":  sqlfile,
	}).Info("exec")

	res, err = h.db.Exec(fmt.Sprintf(string(sqlscript), u.MefeUnitID, h.Code))
	return res, err
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
			response.InternalServerError(w, err.Error())
			return
		}
		ctx.Info("ran unit_disable_existing.sql")
	}
	response.OK(w, metas)
}

func (h handler) createUnit(w http.ResponseWriter, r *http.Request) {

	if r.Body == nil {
		response.BadRequest(w, "Empty payload")
		return
	}

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

		ProductID, err := h.getProductID(unit.MefeUnitID)
		if err == nil {
			results = append(results, ProductID)
			continue
		}

		err = h.step1Insert(unit)
		if err != nil {
			ctx.WithError(err).Error("failed to run step1Insert")
			response.InternalServerError(w, err.Error())
			return
		}

		ctx.Info("inserted")

		start := time.Now()
		_, err = h.runsqlUnit("unit_create_new.sql", unit)
		if err != nil {
			ctx.WithError(err).Errorf("unit_create_new.sql failed")
			response.InternalServerError(w, err.Error())
			return
		}

		ctx.WithField("duration", time.Since(start).String()).Infof("ran unit_create_new.sql")
		ProductID, err = h.getProductID(unit.MefeUnitID)
		if err != nil {
			ctx.WithError(err).Errorf("unit_create_new.sql failed")
			response.InternalServerError(w, err.Error())
			return
		}
		results = append(results, ProductID)
	}
	log.Infof("results: %#v", results)
	response.OK(w, results)

}

func (h handler) getProductID(MefeUnitID string) (newUnit unitCreated, err error) {
	err = h.db.QueryRow("SELECT product_id FROM ut_data_to_create_units WHERE mefe_unit_id=?", MefeUnitID).
		Scan(&newUnit.ProductID)
	if err != nil {
		return newUnit, err
	}
	err = h.db.QueryRow("SELECT name FROM products WHERE id=?", newUnit.ProductID).
		Scan(&newUnit.UnitName)
	return newUnit, err
}
