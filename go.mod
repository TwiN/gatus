module github.com/TwinProduction/gatus

go 1.16

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/TwinProduction/gocache v1.2.3
	github.com/TwinProduction/health v1.0.0
	github.com/go-ping/ping v0.0.0-20201115131931-3300c582a663
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/miekg/dns v1.1.35
	github.com/prometheus/client_golang v1.9.0
	github.com/wcharczuk/go-chart/v2 v2.1.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/image v0.0.0-20210628002857-a66eb6448b8d // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.14
	k8s.io/apimachinery v0.18.14
	k8s.io/client-go v0.18.14
	modernc.org/sqlite v1.11.2
)

replace k8s.io/client-go => k8s.io/client-go v0.18.14
