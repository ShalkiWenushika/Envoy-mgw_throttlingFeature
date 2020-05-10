package filters

import (
	"envoy-test-filter/dtos"
	"sync"
)

var mtx = &sync.Mutex{}
var ch chan map[string]ThrottleData

func InitThrottleDataReceiver() {
	InitiateThrottleCounters()
	InitiateCleanUpTask()
}

func InitiateThrottleCounters() {
	apiLevelCounter := make(map[string]ThrottleData)
	resourceLevelCounter := make(map[string]ThrottleData)
	applicationLevelCounter := make(map[string]ThrottleData)
	subscriptionLevelCounter := make(map[string]ThrottleData)
	ch = make(chan map[string]ThrottleData, 4)
	updateToChannel(ch, apiLevelCounter, resourceLevelCounter, applicationLevelCounter, subscriptionLevelCounter)
}

func updateToChannel(ch chan map[string]ThrottleData, apiLevelCounter map[string]ThrottleData,
	resourceLevelCounter map[string]ThrottleData, applicationLevelCounter map[string]ThrottleData,
	subscriptionLevelCounter map[string]ThrottleData) {
	ch <- apiLevelCounter
	ch <- resourceLevelCounter
	ch <- applicationLevelCounter
	ch <- subscriptionLevelCounter
}

func getMutex() *sync.Mutex {
	return mtx
}

func getChannel() chan map[string]ThrottleData {
	return ch
}

func PublishNonThrottleEvent(throttleEvent dtos.RequestStreamDTO) {
	//Publish throttle event to internal policies
	enabledGlobalTMEventPublishing := false
	if !enabledGlobalTMEventPublishing {
		ProcessNonThrottledEvent(throttleEvent)
	}
}

//This method used to pass throttle data and let it run within separate goroutine
func ProcessNonThrottledEvent(throttleEvent dtos.RequestStreamDTO) {
	mutex := getMutex()
	channel := getChannel()
	setDataReference(throttleEvent)
	go run(mutex, channel)
}
