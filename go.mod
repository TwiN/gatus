module github.com/TwinProduction/gatus

go 1.15

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/TwinProduction/gocache v1.2.1
	github.com/go-ping/ping v0.0.0-20201115131931-3300c582a663
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/miekg/dns v1.1.35
	github.com/prometheus/client_golang v1.9.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	golang.org/x/sys v0.0.0-20201223074533-0d417f636930 // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.18.14
	k8s.io/apimachinery v0.18.14
	k8s.io/client-go v0.18.14
)

replace k8s.io/client-go => k8s.io/client-go v0.18.14
