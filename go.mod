module github.com/unee-t/unit

go 1.12

require (
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/apex/log v1.1.0
	github.com/aws/aws-sdk-go-v2 v0.9.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gorilla/mux v1.7.2
	github.com/pkg/errors v0.8.1 // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.6.0 // indirect
	github.com/tj/assert v0.0.0-20171129193455-018094318fb0 // indirect
	github.com/tj/go v1.8.6
	github.com/unee-t/env v0.0.0-20190513035325-a55bf10999d5
	golang.org/x/sys v0.0.0-20190621203818-d432491b9138 // indirect
	google.golang.org/appengine v1.6.1 // indirect
)

replace github.com/aws/aws-sdk-go-v2 => github.com/aws/aws-sdk-go-v2 v0.7.0
