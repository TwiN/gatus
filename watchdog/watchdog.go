package watchdog

import (
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"sync"
	"time"
)

var (
	serviceResults = make(map[string][]*core.Result)
	rwLock         sync.RWMutex
)

func GetServiceResults() *map[string][]*core.Result {
	return &serviceResults
}

func Monitor() {
	for {
		for _, service := range config.Get().Services {
			go func(service *core.Service) {
				result := service.EvaluateConditions()
				rwLock.Lock()
				defer rwLock.Unlock()
				serviceResults[service.Name] = append(serviceResults[service.Name], result)
				if len(serviceResults[service.Name]) > 15 {
					serviceResults[service.Name] = serviceResults[service.Name][15:]
				}
			}(service)
		}
		time.Sleep(10 * time.Second)
	}
}
