package filters

import (
	"envoy-test-filter/dtos"
)

var apiLevelCounter map[string]ThrottleData
var resourceLevelCounter map[string]ThrottleData
var applicationLevelCounter map[string]ThrottleData
var subscriptionLevelCounter map[string]ThrottleData

func InitThrottleDataReceiver() {
	InitiateThrottleCounters()
	InitiateCleanUpTask()
}

func InitiateThrottleCounters() {
	apiLevelCounter = make(map[string]ThrottleData)
	resourceLevelCounter = make(map[string]ThrottleData)
	applicationLevelCounter = make(map[string]ThrottleData)
	subscriptionLevelCounter = make(map[string]ThrottleData)
}

func getThrottleCounters() (map[string]ThrottleData, map[string]ThrottleData, map[string]ThrottleData,
	map[string]ThrottleData) {
	return apiLevelCounter, resourceLevelCounter, applicationLevelCounter, subscriptionLevelCounter
}

//This method used to pass throttle data and let it run within separate goroutine
func processNonThrottledEvent(throttleEvent dtos.RequestStreamDTO) {
	setDataReference(throttleEvent)
	go run()
}
