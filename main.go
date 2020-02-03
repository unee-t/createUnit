package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tj/go/http/response"

	//	"github.com/unee-t/env"

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
	Code           EnvCode
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

// Debugging - This is the code that should be in uneet/env

// Env is how we manage our differing {dev,demo,prod} AWS accounts

type Env struct {
	Code      EnvCode
	Cfg       aws.Config
	AccountID string
	Stage     string
}

type EnvCode int

// https://github.com/unee-t/processInvitations/blob/master/sql/1_process_one_invitation_all_scenario_v3.0.sql#L12-L16
const (
	EnvUnknown EnvCode = iota // Oops
	EnvDev                    // Development aka Staging
	EnvProd                   // Production
	EnvDemo                   // Demo, which is like Production, for prospective customers to try
)

// GetSecret is the Golang equivalent for
// aws --profile uneet-dev ssm get-parameters --names API_ACCESS_TOKEN --with-decryption --query Parameters[0].Value --output text

func (e Env) GetSecret(key string) string {

	val, ok := os.LookupEnv(key)
	if ok {
		log.Warnf("%s overridden by local env: %s", key, val)
		return val
	}
	// Ideally environment above is set to avoid costly ssm (parameter store) lookups

	ps := ssm.New(e.Cfg)
	in := &ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	}
	req := ps.GetParameterRequest(in)
	out, err := req.Send(context.TODO())
	if err != nil {
		log.WithError(err).Errorf("failed to retrieve credentials for looking up %s", key)
		return ""
	}
	return aws.StringValue(out.Parameter.Value)
}

// NewConfig setups the configuration assuming various parameters have been setup in the AWS account
// - DEFAULT_REGION
// - STAGE
// -
func NewConfig(cfg aws.Config) (e Env, err error) {

	defaultRegion, ok := os.LookupEnv("DEFAULT_REGION")
	// the AWS variable `DEFAULT_REGION` is in the format `ap-southeast-1`
	// We can use the repo https://github.com/aws/aws-sdk-go/ to convert this to a format like `ApSoutheast1RegionID`
	// TODO - Check with @kai if the format `ap-southeast-1` is OK or if we need to transform that...
	if !ok {
		defaultRegion = endpoints.ApSoutheast1RegionID
	}

	cfg.Region = defaultRegion
	log.Warnf("Env Region: %s", cfg.Region)

	// Save for ssm
	e.Cfg = cfg

	svc := sts.New(cfg)
	input := &sts.GetCallerIdentityInput{}
	req := svc.GetCallerIdentityRequest(input)
	result, err := req.Send(context.TODO())
	if err != nil {
		return e, err
	}

	e.AccountID = aws.StringValue(result.Account)
	log.Infof("Account ID: %s", result.Account)

	e.Stage = e.GetSecret("STAGE")

	switch e.Stage {
	case "dev":
		e.Code = EnvDev
		return e, nil
	case "prod":
		e.Code = EnvProd
		return e, nil
	case "demo":
		e.Code = EnvDemo
		return e, nil
	default:
		log.WithField("stage", e.Stage).Error("unknown stage")
		return e, nil
	}
}

func (e Env) Bucket(svc string) string {
	// Most common bucket
	if svc == "" {
		svc = "media"
	}
	installationID := e.GetSecret("INSTALLATION_ID")
	if installationID == "" {
		installationID = "main"
		log.Warnf("Using fallback INSTALLATION_ID: %s: ", installationID)
	}
	if installationID == "main" {
		// Preserve original bucket names
		return fmt.Sprintf("%s-%s-unee-t", e.Stage, svc)
	} else {
		// Use INSTALLATION_ID to generate unique bucket name
		return fmt.Sprintf("%s-%s-%s", e.Stage, svc, installationID)
	}
}

func (e Env) SNS(name, region string) string {
	if name == "" {
		log.Warn("Service string empty")
		return ""
	}
	return fmt.Sprintf("arn:aws:sns:%s:%s:%s", region, e.AccountID, name)
}

func (e Env) Udomain(service string) string {
	if service == "" {
		log.Warn("Service string empty")
		return ""
	}
	domain := e.GetSecret("DOMAIN")
	if domain == "" {
		domain = "unee-t.com"
		log.Warnf("Using fallback domain: %s: ", domain)
	}
	switch e.Code {
	case EnvDev:
		return fmt.Sprintf("%s.dev.%s", service, domain)
	case EnvProd:
		return fmt.Sprintf("%s.%s", service, domain)
	case EnvDemo:
		return fmt.Sprintf("%s.demo.%s", service, domain)
	default:
		log.Warnf("Udomain warning: Env %d is unknown, resorting to dev", e.Code)
		return fmt.Sprintf("%s.dev.unee-t.com", service)
	}
}

func (e Env) BugzillaDSN() string {
	var bugzillaDbUser string
	valbugzillaDbUser, ok := os.LookupEnv("BUGZILLA_DB_USER")
	if ok {
		log.Infof("BUGZILLA_DB_USER overridden by local env: %s", valbugzillaDbUser)
		bugzillaDbUser = valbugzillaDbUser
	} else {
		bugzillaDbUser = e.GetSecret("BUGZILLA_DB_USER")
	}

	if bugzillaDbUser == "" {
		log.Fatal("BUGZILLA_DB_USER is unset")
	}

	var mysqlhost string
	val, ok := os.LookupEnv("MYSQL_HOST")
	if ok {
		log.Infof("MYSQL_HOST overridden by local env: %s", val)
		mysqlhost = val
	} else {
		mysqlhost = e.GetSecret("MYSQL_HOST")
	}

	if mysqlhost == "" {
		log.Fatal("MYSQL_HOST is unset")
	}

	return fmt.Sprintf("%s:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL&timeout=5s&collation=utf8mb4_unicode_520_ci",
		bugzillaDbUser,
		e.GetSecret("BUGZILLA_DB_PASSWORD"),
		mysqlhost)
}

// Protect using: curl -H 'Authorization: Bearer secret' style
// Modelled after https://github.com/apex/up-examples/blob/master/oss/golang-basic-auth/main.go#L16
func Protect(h http.Handler, APIAccessToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		// Get token from the Authorization header
		// format: Authorization: Bearer
		tokens, ok := r.Header["Authorization"]
		if ok && len(tokens) >= 1 {
			token = tokens[0]
			token = strings.TrimPrefix(token, "Bearer ")
		}
		if token == "" || token != APIAccessToken {
			log.Errorf("Token %q != APIAccessToken %q", token, APIAccessToken)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Towr is a workaround for gorilla/pat: https://stackoverflow.com/questions/50753049/
// Wish I could make this simpler
func Towr(h http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }
}

// END Debugging

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

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("uneet-dev"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID
	e, err := NewConfig(cfg)
	if err != nil {
		log.WithError(err).Warn("error getting unee-t env")
	}

	h = handler{
		DSN:            e.BugzillaDSN(), // `BugzillaDSN` is a function that is defined in the uneet/env/main.go dependency.
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
	return Protect(app, h.APIAccessToken)
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
