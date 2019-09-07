package main

import (
	"fmt"
	"github.com/TwinProduction/gatus/config"
)

func main() {
	for _, service := range config.Get().Services {
		result := service.EvaluateConditions()
		for _, conditionResult := range result.ConditionResult {
			fmt.Printf("%v\n", *conditionResult)
		}
	}
}
