package filters

import (
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

func updateCounters(apiKey string, appKey string, stopOnQuota bool, subscriptionKey string, appTierCount int64,
	appTierUnitTime int64, appTierTimeUnit string, apiTierCount int64, apiTierUnitTime int64, apiTierTimeUnit string,
	subscriptionTierCount int64, subscriptionTierUnitTime int64, subscriptionTierTimeUnit string, resourceKey string,
	resourceTierCount int64, resourceTierUnitTime int64, resourceTierTimeUnit string, timestamp int64, mtx *sync.Mutex,
	ch chan map[string]ThrottleData) {
	apiLevelCounter := <-ch
	resourceLevelCounter := <-ch
	applicationLevelCounter := <-ch
	subscriptionLevelCounter := <-ch
	updateMapCounters(apiLevelCounter, apiKey, stopOnQuota, apiTierCount, apiTierUnitTime, apiTierTimeUnit, timestamp,
		ThrottleType(3), mtx)
	updateMapCounters(resourceLevelCounter, resourceKey, stopOnQuota, resourceTierCount, resourceTierUnitTime,
		resourceTierTimeUnit, timestamp, ThrottleType(2), mtx)
	updateMapCounters(applicationLevelCounter, appKey, stopOnQuota, appTierCount, appTierUnitTime, appTierTimeUnit,
		timestamp, ThrottleType(0), mtx)
	updateMapCounters(subscriptionLevelCounter, subscriptionKey, stopOnQuota, subscriptionTierCount,
		subscriptionTierUnitTime, subscriptionTierTimeUnit, timestamp, ThrottleType(2), mtx)
	updateToChannel(ch, apiLevelCounter, resourceLevelCounter, applicationLevelCounter, subscriptionLevelCounter)
}

func updateMapCounters(counterMap map[string]ThrottleData, throttleKey string, stopOnQuota bool, limit int64,
	unitTime int64, timeUnit string, timestamp int64, throttleType ThrottleType, mtx *sync.Mutex) {
	throttleData, found := counterMap[throttleKey]
	if found {
		count := throttleData.getCount() + 1
		if limit > 0 && count >= limit {
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
		log.Infof("Throttle count for the key %v is %v", throttleKey, throttleData.getCount())
		mtx.Lock()
		counterMap[throttleKey] = throttleData
		mtx.Unlock()
	} else {
		var throttleData ThrottleData = ThrottleData{}
		var startTime int64 = timestamp - (timestamp % getTimeInMilliSeconds(1, timeUnit))
		throttleData.setWindowStartTime(startTime)
		throttleData.setStopOnQuota(stopOnQuota)
		throttleData.setUnitTime(getTimeInMilliSeconds(unitTime, timeUnit))
		throttleData.setThrottleType(throttleType)
		throttleData.setCount(0)
		throttleData.setThrottleKey(throttleKey)
		addThrottleData(throttleData)
		mtx.Lock()
		counterMap[throttleKey] = throttleData
		mtx.Unlock()
	}

}

func getApiLevelCounter() map[string]ThrottleData {
	ch := getChannel()
	apiLevelCounter := <-ch
	return apiLevelCounter
}

func getResourceLevelCounter() map[string]ThrottleData {
	ch := getChannel()
	<-ch
	resourceLevelCounter := <-ch
	return resourceLevelCounter
}

func getApplicationLevelCounter() map[string]ThrottleData {
	ch := getChannel()
	<-ch
	<-ch
	<-ch
	applicationLevelCounter := <-ch
	return applicationLevelCounter
}

func getSubscriptionLevelCounter() map[string]ThrottleData {
	ch := getChannel()
	<-ch
	<-ch
	<-ch
	<-ch
	subscriptionLevelCounter := <-ch
	return subscriptionLevelCounter
}

func isApiLevelThrottled(apiKey string) bool {
	apiLevelCounter := getApiLevelCounter()
	return isRequestThrottled(apiLevelCounter, apiKey)
}

func isResourceThrottled(resourceKey string) bool {
	resourceLevelCounter := getResourceLevelCounter()
	return isRequestThrottled(resourceLevelCounter, resourceKey)
}

func isAppLevelThrottled(appKey string) bool {
	applicationLevelCounter := getApplicationLevelCounter()
	return isRequestThrottled(applicationLevelCounter, appKey)
}

func isSubsLevelThrottled(subscriptionKey string) bool {
	subscriptionLevelCounter := getSubscriptionLevelCounter()
	return isRequestThrottled(subscriptionLevelCounter, subscriptionKey)
}

func removeFromResourceCounterMap(key string) {
	resourceLevelCounter := getResourceLevelCounter()
	delete(resourceLevelCounter, key)
}

func removeFromApplicationCounterMap(key string) {
	applicationLevelCounter := getApplicationLevelCounter()
	delete(applicationLevelCounter, key)
}

func removeFromApiCounterMap(key string) {
	apiLevelCounter := getApiLevelCounter()
	delete(apiLevelCounter, key)
}

func removeFromSubscriptionCounterMap(key string) {
	subscriptionLevelCounter := getSubscriptionLevelCounter()
	delete(subscriptionLevelCounter, key)
}

func isRequestThrottled(counterMap map[string]ThrottleData, throttleKey string) bool {
	if _, found := counterMap[throttleKey]; found {
		var currentTime int64 = getCurrentTimeMillis()
		var throttleData ThrottleData = counterMap[throttleKey]
		if currentTime > throttleData.getWindowStartTime()+throttleData.getUnitTime() {
			throttleData.setThrottled(false)
			counterMap[throttleKey] = throttleData
			log.Warnf("Throttle window has expired. CurrentTime : %v \n Window start time : %v \n Unit time : ",
				currentTime, throttleData.getWindowStartTime(), throttleData.getUnitTime())
			return false
		}
		return throttleData.isThrottled()
	}
	return false
}

func getCurrentTimeMillis() int64 {
	now := time.Now()
	unixNano := now.UnixNano()
	umillisec := unixNano / 1000000
	//fmt.Println("Current time in millis> ", umillisec)
	return umillisec
}

func getTimeInMilliSeconds(unitTime int64, timeUnit string) int64 {
	var milliSeconds int64
	if strings.EqualFold("min", timeUnit) {
		milliSeconds = time.Minute.Milliseconds() * unitTime
	} else if strings.EqualFold("hour", timeUnit) {
		milliSeconds = time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("day", timeUnit) {
		milliSeconds = 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("week", timeUnit) {
		milliSeconds = 7 * 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("month", timeUnit) {
		milliSeconds = 30 * 24 * time.Hour.Milliseconds() * unitTime
	} else if strings.EqualFold("year", timeUnit) {
		milliSeconds = 365 * 24 * time.Hour.Milliseconds() * unitTime
	} else {
		log.Warnf("Unsupported time unit provided")
	}
	//fmt.Println("ms: ",milliSeconds)
	return milliSeconds
}
