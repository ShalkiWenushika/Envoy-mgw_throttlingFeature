package filters

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

//var apiLevelCounter = make(map[string]ThrottleData)
//var resourceLevelCounter = make(map[string]ThrottleData)
//var applicationLevelCounter = make(map[string]ThrottleData)
//var subscriptionLevelCounter = make(map[string]ThrottleData)

func updateCounters(apiKey string, appKey string, stopOnQuota bool, subscriptionKey string, appTierCount int64,
	appTierUnitTime int64, appTierTimeUnit string, apiTierCount int64, apiTierUnitTime int64, apiTierTimeUnit string,
	subscriptionTierCount int64, subscriptionTierUnitTime int64, subscriptionTierTimeUnit string, resourceKey string,
	resourceTierCount int64, resourceTierUnitTime int64, resourceTierTimeUnit string, timestamp int64){
	apiLevelCounter, resourceLevelCounter, applicationLevelCounter, subscriptionLevelCounter = getThrottleCounters()
	updateMapCounters(apiLevelCounter, apiKey, stopOnQuota, apiTierCount, apiTierUnitTime, apiTierTimeUnit, timestamp, 
		ThrottleType(3))
}

func updateMapCounters(counterMap map[string]ThrottleData, throttleKey string, stopOnQuota bool, limit int64,
	unitTime int64, timeUnit string, timestamp int64, throttleType ThrottleType){
	throttleData,found := counterMap[throttleKey]
	if found {
		count := throttleData.getCount() + 1
		if limit>0 && count >= limit {
			throttleData.setThrottled(true)
		} else {
			throttleData.setThrottled(false)
		}
		if timestamp > (throttleData.getWindowStartTime() + throttleData.getUnitTime()) {
			throttleData.setCount(1)
			var startTime int64 = timestamp - (timestamp % getTimeInMilliSeconds(1, timeUnit))
			throttleData.setWindowStartTime(startTime)
			throttleData.setThrottled(false)
		}
		log.Infof("Throttle count for the key %v is %v" ,throttleKey, throttleData.getCount())
	} else {
		var throttleData ThrottleData = ThrottleData{}
		var startTime int64 = timestamp - (timestamp % getTimeInMilliSeconds(1, timeUnit))
		throttleData.setWindowStartTime(startTime)
		throttleData.setStopOnQuota(stopOnQuota)
		throttleData.setUnitTime(getTimeInMilliSeconds(unitTime, timeUnit))
		throttleData.setThrottleType(throttleType)
		throttleData.setCount(0)
		throttleData.setThrottleKey(throttleKey)
		counterMap[throttleKey] = throttleData
	}

}

func isApiLevelThrottled(apiKey string) bool {
	return isRequestThrottled(apiLevelCounter, apiKey)
}

func isResourceThrottled(resourceKey string) bool {
	return isRequestThrottled(resourceLevelCounter, resourceKey)
}

func isAppLevelThrottled(appKey string) bool {
	return isRequestThrottled(applicationLevelCounter, appKey)
}

func isSubsLevelThrottled(subscriptionKey string) bool {
	return isRequestThrottled(subscriptionLevelCounter, subscriptionKey)
}

func K(apiKey string) bool {
	return isRequestThrottled(apiLevelCounter, apiKey)
}

func removeFromResourceCounterMap(key string) {
	delete(resourceLevelCounter, key)
}

func removeFromApplicationCounterMap(key string){
	delete(applicationLevelCounter, key)
}

func removeFromApiCounterMap(key string){
	delete(apiLevelCounter, key)
}

func removeFromSubscriptionCounterMap(key string){
	delete(subscriptionLevelCounter, key)
}

func isRequestThrottled(counterMap map[string]ThrottleData, throttleKey string) bool {
	if _,found := counterMap[throttleKey];found {
		var currentTime int64 = getCurrentTimeMillis()
		var throttleData ThrottleData = counterMap[throttleKey]
		if currentTime > throttleData.getWindowStartTime() + throttleData.getUnitTime() {
			throttleData.setThrottled(false)
			counterMap[throttleKey] = throttleData
			log.Warnf("Throttle window has expired. CurrentTime : %v \n Window start time : %v \n Unit time : ",
			currentTime, throttleData.getWindowStartTime(), throttleData.getUnitTime() )
			return false
		}
		return throttleData.isThrottled()
	}
	return false
}

func getCurrentTimeMillis() int64 {
	now := time.Now()
	unixNano := now.UnixNano()
	umillisec := unixNano/1000000
	//fmt.Println("Current time in millis> ", umillisec)
	return umillisec
}

func getTimeInMilliSeconds(unitTime int64, timeUnit string) int64{
	var milliSeconds int64
	if strings.EqualFold("min",timeUnit) {
		milliSeconds = time.Minute.Milliseconds() * unitTime
	} else if strings.EqualFold("hour",timeUnit) {
		milliSeconds = time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("day",timeUnit) {
		milliSeconds = 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("week",timeUnit){
		milliSeconds = 7 * 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("month",timeUnit){
		milliSeconds = 30 * 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("year",timeUnit){
		milliSeconds = 365 * 24 * time.Hour.Milliseconds() * unitTime
	} else {
		log.Warnf("Unsupported time unit provided")
	}
	//fmt.Println("ms: ",milliSeconds)
	return milliSeconds
}
