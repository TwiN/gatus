package store

import (
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage/store/memory"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	timestamp = time.Now()

	testService = core.Service{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []*core.Condition{&firstCondition, &secondCondition, &thirdCondition},
		Alerts:                  nil,
		Insecure:                false,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Body:                  []byte("body"),
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             timestamp,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   true,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   true,
			},
		},
	}
	testUnsuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Body:                  []byte("body"),
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             timestamp,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   false,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   false,
			},
		},
	}
)

func BenchmarkStore_GetAllAsJSON(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
	}
	for _, scenario := range scenarios {
		scenario.Store.Insert(&testService, &testSuccessfulResult)
		scenario.Store.Insert(&testService, &testUnsuccessfulResult)
		b.Run(scenario.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				scenario.Store.GetAllAsJSON()
			}
			b.ReportAllocs()
		})
	}
}

func BenchmarkStore_Insert(b *testing.B) {
	memoryStore, err := memory.NewStore("")
	if err != nil {
		b.Fatal("failed to create store:", err.Error())
	}
	type Scenario struct {
		Name  string
		Store Store
	}
	scenarios := []Scenario{
		{
			Name:  "memory",
			Store: memoryStore,
		},
	}
	for _, scenario := range scenarios {
		b.Run(scenario.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				scenario.Store.Insert(&testService, &testSuccessfulResult)
				scenario.Store.Insert(&testService, &testUnsuccessfulResult)
			}
			b.ReportAllocs()
		})
	}
}
