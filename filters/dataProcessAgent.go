package filters

import "envoy-test-filter/dtos"

var apiKey string
var appKey string
var stopOnQuota bool
var subscriptionKey string
var appTierCount int64
var appTierUnitTime int64
var appTierTimeUnit string
var apiTierCount int64
var apiTierUnitTime int64
var apiTierTimeUnit string
var subscriptionTierCount int64
var subscriptionTierUnitTime int64
var subscriptionTierTimeUnit string
var resourceKey string
var resourceTierCount int64
var resourceTierUnitTime int64
var resourceTierTimeUnit string
var timestamp int64

func setDataReference (throttleData dtos.RequestStreamDTO) {
	appKey = throttleData.AppKey
	appTierCount = throttleData.AppTierCount
	appTierUnitTime = throttleData.AppTierUnitTime
	appTierTimeUnit = throttleData.AppTierTimeUnit
	apiKey = throttleData.ApiKey
	apiTierCount = throttleData.AppTierCount
	apiTierUnitTime = throttleData.AppTierUnitTime
	apiTierTimeUnit = throttleData.ApiTierTimeUnit
	subscriptionKey = throttleData.SubscriptionKey
	subscriptionTierCount = throttleData.SubscriptionTierCount
	subscriptionTierUnitTime = throttleData.SubscriptionTierUnitTime
	subscriptionTierTimeUnit = throttleData.SubscriptionTierTimeUnit
	resourceKey = throttleData.ResourceKey
	resourceTierCount = throttleData.ResourceTierCount
	resourceTierUnitTime = throttleData.ResourceTierUnitTime
	resourceTierTimeUnit = throttleData.ResourceTierTimeUnit
	stopOnQuota = throttleData.StopOnQuota
	timestamp = getCurrentTimeMillis()
}

func run () {
	updateCounters(apiKey, appKey, stopOnQuota, subscriptionKey, appTierCount, appTierUnitTime,
		appTierTimeUnit, apiTierCount, apiTierUnitTime, apiTierTimeUnit, subscriptionTierCount,
		subscriptionTierUnitTime, subscriptionTierTimeUnit, resourceKey, resourceTierCount,
		resourceTierUnitTime, resourceTierTimeUnit, timestamp)
}