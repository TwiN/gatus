package main

import (
	"fmt"
	"github.com/TwinProduction/gatus/watchdog"
)

func main() {
	request := watchdog.Request{Url: "https://twinnation.org/actuator/health"}
	result := &watchdog.Result{}
	request.GetIp(result)
	request.GetStatus(result)
	fmt.Println(result)
}
