package provider

import (
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

type AlertProvider interface {
	IsValid() bool
	ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider
}
