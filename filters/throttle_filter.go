package filters

import (
	"context"
	"envoy-test-filter/Constants"
	"envoy-test-filter/dtos"
	"fmt"
	_ "github.com/cactus/go-statsd-client/statsd"
	ext_authz "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	"github.com/gogo/googleapis/google/rpc"
	log "github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"
	_ "log"
	"reflect"
	"strconv"
)

var enabledGlobalTMEventPublishing bool = false

func ThrottleFilter(ctx context.Context, req *ext_authz.CheckRequest) (*ext_authz.CheckResponse, error) {

	fmt.Println("Processing the request in ThrottleFilter>>>>>>>>>>>>>>")
	fmt.Println("Context: ", ctx)
	deployedPolicies := getDeployedPolicies()
	//fmt.Println("deployed policies: ", deployedPolicies)
	doThrottleFilterRequest(ctx, deployedPolicies)
	//shalki()
	//getApiLevelCounter()
	//getTimeInMilliSeconds(1,"min")
	//getCurrentTimeMillis()
	//getType()

	resp := &ext_authz.CheckResponse{}
	resp = &ext_authz.CheckResponse{
		Status: &status.Status{Code: int32(rpc.OK)},
		HttpResponse: &ext_authz.CheckResponse_OkResponse{
			OkResponse: &ext_authz.OkHttpResponse{},
		},
	}
	return resp, nil
}

func getDeployedPolicies() map[string]map[string]string {
	deployedPolicies := map[string]map[string]string{
		"app_50PerMin": map[string]string{
			"count":       "50",
			"unitTime":    "1",
			"timeUnit":    "min",
			"stopOnQuota": "true",
		},
		"app_20PerMin": map[string]string{
			"count":       "20",
			"unitTime":    "1",
			"timeUnit":    "min",
			"stopOnQuota": "true",
		},
		"res_10PerMin": map[string]string{
			"count":       "10",
			"unitTime":    "1",
			"timeUnit":    "min",
			"stopOnQuota": "true",
		},
	}
	//aaa
	return deployedPolicies
}

func getInvocationContext() map[string]string {
	invocationContext := map[string]string{
		"AUTHENTICATION_CONTEXT": "authenticated",
		"IS_SECURED":             "false",
		"KEY_TYPE":               "PRODUCTION",
	}
	return invocationContext
}

func getKeyValidationResult() map[string]string {
	keyValidationResult := map[string]string{
		"authenticated":   "true",
		"username":        "admin",
		"applicationTier": "Unlimited",
		"tier":            "Default",
		"apiTier":         "Unlimited",
		"applicationId":   "899",
	}
	return keyValidationResult
}

func doThrottleFilterRequest(ctx context.Context, deployedPolicies map[string]map[string]string) bool {
	invocationContext := getInvocationContext()
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Processing the request in ThrottleFilter")
	//Throttled decisions
	var isThrottled bool = false
	var stopOnQuota bool
	isSecured, err := strconv.ParseBool(invocationContext[Constants.IS_SECURED])
	if err != nil {
		log.Println(err)
	}
	var apiVersion string = "1.0.0"
	var resourceLevelPolicyName string = "50PerMin"
	keyValidationResult := map[string]string{}
	if _, found := invocationContext[Constants.AUTHENTICATION_CONTEXT]; found {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Context contains Authentication Context")
		keyValidationResult = getKeyValidationResult()
		var apiLevelPolicy string = "10PerMin"
		if !checkAPILevelThrottled(ctx, apiLevelPolicy, apiVersion, deployedPolicies) {
			return false
		}
		if !checkResourceLevelThrottled(ctx, resourceLevelPolicyName, apiVersion, deployedPolicies) {
			return false
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking subscription level throttle policy " +
			keyValidationResult[Constants.TIER] + " exist.")
		if keyValidationResult[Constants.TIER] != Constants.UNLIMITED_TIER && !isPolicyExist(deployedPolicies,
			keyValidationResult[Constants.TIER], Constants.SUB_LEVEL_PREFIX) {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Subscription level throttle policy " +
				keyValidationResult[Constants.TIER] + " does not exist.")
			//setThrottleErrorMessageToContext
			//sendErrorResponse
			return false
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking subscription level throttling-out.")
		isThrottled, stopOnQuota = isSubscriptionLevelThrottled(ctx, keyValidationResult, deployedPolicies, apiVersion)
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Subscription level throttling result:: isThrottled: " +
			strconv.FormatBool(isThrottled) + " , stopOnQuota: " + strconv.FormatBool(stopOnQuota))
		if isThrottled {
			if stopOnQuota {
				log.Debugf(Constants.KEY_THROTTLE_FILTER + "Sending throttled out responses.")
				//set context
				//setThrottleErrorMessageToContex
				//sendErrorResponse
				return false
			} else {
				// set properties in order to publish into analytics for billing
				log.Debugf(Constants.KEY_THROTTLE_FILTER + "Proceeding(1st) since stopOnQuota is set to false.")
			}
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking application level throttle policy " +
			keyValidationResult[Constants.APPLICATION_TIER] + " exist.")
		if keyValidationResult[Constants.APPLICATION_TIER] != Constants.UNLIMITED_TIER &&
			!isPolicyExist(deployedPolicies, keyValidationResult[Constants.APPLICATION_TIER],
				Constants.APP_LEVEL_PREFIX) {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Application level throttle policy " +
				keyValidationResult[Constants.APPLICATION_TIER] + " does not exist.")
			//setThrottleErrorMessageToContext
			//sendErrorResponse
			return false
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking application level throttling-out.")
		if isApplicationLevelThrottled(keyValidationResult, deployedPolicies) {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Application level throttled out. Sending throttled " +
				"out response.")
			//set context attributes
			//setThrottleErrorMessageToContext
			//sendErrorResponse
			return false
		} else {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Application level throttled out: false")
		}
	} else if !isSecured {
		var apiLevelPolicy string = "10PerMin"
		if !checkAPILevelThrottled(ctx, apiLevelPolicy, apiVersion, deployedPolicies) {
			return false
		}
		if !checkResourceLevelThrottled(ctx, resourceLevelPolicyName, apiVersion, deployedPolicies) {
			return false
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Not a secured resource. Proceeding with Unauthenticated tier.")
		// setting keytype to invocationContext
		invocationContext[Constants.KEY_TYPE_ATTR] = Constants.PRODUCTION_KEY_TYPE

		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking unauthenticated throttle policy  " +
			Constants.UNAUTHENTICATED_TIER + "  exist.")
		if !isPolicyExist(deployedPolicies, Constants.UNAUTHENTICATED_TIER, Constants.SUB_LEVEL_PREFIX) {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Unauthenticated throttle policy " +
				Constants.UNAUTHENTICATED_TIER + "  does not exist.")
			//setThrottleErrorMessageToContext
			//sendErrorResponse
			return false
		}
		//[isThrottled, stopOnQuota] = isUnauthenticateLevelThrottled(ctx)
	} else {
		log.Debugf("Unknown error.")
		//setThrottleErrorMessageToContext
		//sendErrorResponse
		return false
	}
	//Publish throttle event to another worker flow to publish to internal policies or traffic manager
	var throttleEvent dtos.RequestStreamDTO = generateThrottleEvent(ctx, keyValidationResult, deployedPolicies)
	publishEvent(throttleEvent)

	return true
}

func checkAPILevelThrottled(ctx context.Context, apiLevelPolicy string, apiVersion string,
	deployedPolicies map[string]map[string]string) bool {
	log.Debugf("Checking api level throttle policy " + apiLevelPolicy + " exist.")
	if apiLevelPolicy != Constants.UNLIMITED_TIER && !isPolicyExist(deployedPolicies, apiLevelPolicy,
		Constants.RESOURCE_LEVEL_PREFIX) {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + ", API level throttle policy " + apiLevelPolicy +
			"does not exist.")
		//setThrottleErrorMessageToContext
		//sendErrorResponse
		return false
	}
	log.Debugf(Constants.KEY_THROTTLE_FILTER + ", Checking API level throttling-out.")
	if isAPILevelThrottled(ctx, apiVersion) {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "API level throttled out. Sending throttled out response.")
		//set context attributes
		//setThrottleErrorMessageToContext
		//sendErrorResponse
		return false
	} else {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "API level throttled out: false")
	}
	return true
}

func checkResourceLevelThrottled(ctx context.Context, resourceLevelPolicyName string, apiVersion string,
	deployedPolicies map[string]map[string]string) bool {
	if (reflect.TypeOf(resourceLevelPolicyName).String()) == "string" {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource level throttle policy : " + resourceLevelPolicyName)
		if len(resourceLevelPolicyName) > 0 && resourceLevelPolicyName != Constants.UNLIMITED_TIER &&
			!isPolicyExist(deployedPolicies, resourceLevelPolicyName, Constants.RESOURCE_LEVEL_PREFIX) {
			log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource level throttle policy " +
				resourceLevelPolicyName + " does not exist.")
			//setThrottleErrorMessageToContext
			return false
		}
	}
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking resource level throttling-out.")
	if isResourceLevelThrottled(ctx, resourceLevelPolicyName, deployedPolicies, apiVersion) {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource level throttled out. Sending throttled out response.")
		//set context
		return false
	} else {
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource level throttled out: false")
	}
	return true
}

func isAPILevelThrottled(ctx context.Context, apiVersion string) bool {
	var apiThrottleKey string = "/petstore/v1"
	if (reflect.TypeOf(apiVersion).String()) == "string" {
		apiThrottleKey += ":" + apiVersion
	}
	if enabledGlobalTMEventPublishing {
		apiThrottleKey += "_default"
	}
	if !enabledGlobalTMEventPublishing {
		isApiLevelThrottled(apiThrottleKey)
	}

	return true
}

func isResourceLevelThrottled(ctx context.Context, policy string, deployedPolicies map[string]map[string]string,
	apiVersion string) bool {
	if (reflect.TypeOf(policy).String()) == "string" {
		if policy == Constants.UNLIMITED_TIER {
			return false
		}
		var resourceLevelThrottleKey string = "getc73b6bf1a19545cfa184290e8b65195f"
		if (reflect.TypeOf(apiVersion).String()) == "string" {
			resourceLevelThrottleKey += ":" + apiVersion
		}
		if enabledGlobalTMEventPublishing {
			resourceLevelThrottleKey += "_default"
		}
		log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource level throttle key : " + resourceLevelThrottleKey)
		//var throttled bool
		//var stopOnQuota bool
		if !enabledGlobalTMEventPublishing {
			return isResourceThrottled(resourceLevelThrottleKey)
		}
	}
	return false
}

func isSubscriptionLevelThrottled(ctx context.Context, keyValidationDto map[string]string,
	deployedPolicies map[string]map[string]string, apiVersion string) (bool, bool) {
	contextData := "/petstore/v1"
	if keyValidationDto[Constants.TIER] == Constants.UNLIMITED_TIER {
		return false, false
	}
	var subscriptionLevelThrottleKey string = keyValidationDto[Constants.APPLICATION_ID] + ":" + contextData
	if (reflect.TypeOf(apiVersion).String()) == "string" {
		subscriptionLevelThrottleKey += ":" + apiVersion
	}
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Subscription level throttle key : " + subscriptionLevelThrottleKey)
	if !enabledGlobalTMEventPublishing {
		stopOnQuotaValue := deployedPolicies[Constants.SUB_LEVEL_PREFIX+keyValidationDto[Constants.TIER]]["stopOnQuota"]
		stopOnQuota, err := strconv.ParseBool(stopOnQuotaValue)
		if err == nil {
			var isThrottled bool = isSubsLevelThrottled(subscriptionLevelThrottleKey)
			return isThrottled, stopOnQuota
		}
	}
	return false, false
}

func isApplicationLevelThrottled(keyValidationDto map[string]string,
	deployedPolicies map[string]map[string]string) bool {
	if keyValidationDto[Constants.APPLICATION_TIER] == Constants.UNLIMITED_TIER {
		return false
	}
	var applicationLevelThrottleKey string = keyValidationDto[Constants.APPLICATION_ID] + ":" +
		keyValidationDto[Constants.USERNAME]
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Application level throttle key : " + applicationLevelThrottleKey)
	if !enabledGlobalTMEventPublishing {
		return isAppLevelThrottled(applicationLevelThrottleKey)
	}
	return false
}

func isPolicyExist(deployedPolicies map[string]map[string]string, apiLevelPolicy string, prefix string) bool {
	if _, found := deployedPolicies[prefix+apiLevelPolicy]; found {
		return true
	} else {
		return false
	}
}

func generateThrottleEvent(ctx context.Context, keyValidationDto map[string]string,
	deployedPolicies map[string]map[string]string) dtos.RequestStreamDTO {
	requestStreamDTO := dtos.RequestStreamDTO{}
	if !enabledGlobalTMEventPublishing {
		requestStreamDTO = generateLocalThrottleEvent(ctx, keyValidationDto, deployedPolicies)
	}
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Resource key : " + requestStreamDTO.ResourceKey +
		"Subscription key : " + requestStreamDTO.SubscriptionKey + "App key : " + requestStreamDTO.AppKey +
		"API key : " + requestStreamDTO.ApiKey + "Resource Tier : " + requestStreamDTO.ResourceTier +
		"Subscription Tier : " + requestStreamDTO.SubscriptionTier + "App Tier : " + requestStreamDTO.AppTier +
		"API Tier : " + requestStreamDTO.ApiTier)
	return requestStreamDTO
}

func generateLocalThrottleEvent(ctx context.Context, keyValidationDto map[string]string,
	deployedPolicies map[string]map[string]string) dtos.RequestStreamDTO {
	requestStreamDTO := dtos.RequestStreamDTO{}
	requestStreamDTO = setCommonThrottleData(ctx, keyValidationDto, deployedPolicies)
	requestStreamDTO.AppTierCount = 1
	requestStreamDTO.ApiTierUnitTime = 1
	requestStreamDTO.AppTierTimeUnit = "min"

	requestStreamDTO.SubscriptionTierCount = 1
	requestStreamDTO.SubscriptionTierUnitTime = 1
	requestStreamDTO.SubscriptionTierTimeUnit = "min"
	requestStreamDTO.StopOnQuota = true

	requestStreamDTO.ResourceTierCount = 1
	requestStreamDTO.ResourceTierUnitTime = 1
	requestStreamDTO.ResourceTierTimeUnit = "min"

	requestStreamDTO.ApiTierCount = 1
	requestStreamDTO.ApiTierUnitTime = 1
	requestStreamDTO.ApiTierTimeUnit = "min"
	setThrottleKeysWithVersion(ctx, requestStreamDTO)
	return requestStreamDTO
}

func setThrottleKeysWithVersion(ctx context.Context, requestStreamDTO dtos.RequestStreamDTO) {
	apiVersion := "1.0.0"
	if (reflect.TypeOf(apiVersion).String()) == "string" {
		requestStreamDTO.ApiVersion = apiVersion
		requestStreamDTO.ApiKey += ":" + apiVersion
		requestStreamDTO.SubscriptionKey += ":" + apiVersion
		requestStreamDTO.ResourceKey += ":" + apiVersion
	}
}

func setCommonThrottleData(ctx context.Context, keyValidationDto map[string]string,
	deployedPolicies map[string]map[string]string) dtos.RequestStreamDTO {
	requestStreamDTO := dtos.RequestStreamDTO{ResetTimestamp: 0, RemainingQuota: 0, IsThrottled: false,
		StopOnQuota: true, ResourceTierCount: -1, ResourceTierUnitTime: -1, AppTierCount: -1, AppTierUnitTime: -1,
		ApiTierCount: -1, ApiTierUnitTime: -1, SubscriptionTierCount: -1, SubscriptionTierUnitTime: -1}
	//appPolicyDetails := utils.GetPolicyDetails(deployedPolicies, keyValidationDto[Constants.APPLICATION_TIER],
	//	Constants.APP_LEVEL_PREFIX)
	requestStreamDTO.AppTier = "10MinAppPolicy"
	requestStreamDTO.ApiTier = "10PerMin"
	requestStreamDTO.SubscriptionTier = "10MinSubPolicy"
	requestStreamDTO.ApiKey = "/pizzashack/1.0.0"

	if requestStreamDTO.ApiTier != Constants.UNLIMITED_TIER && requestStreamDTO.ApiTier != "" {
		requestStreamDTO.ResourceTier = requestStreamDTO.ApiTier
		requestStreamDTO.ResourceKey = requestStreamDTO.ApiKey
	} else {
		var resourceKey string = "get87b5e48c92b648e3bba230381d89cfef"
		requestStreamDTO.ResourceTier = "3PerMin"
		requestStreamDTO.ResourceKey = resourceKey
	}

	requestStreamDTO.AppKey = keyValidationDto[Constants.APPLICATION_ID] + ":" + keyValidationDto[Constants.USERNAME]
	requestStreamDTO.SubscriptionKey = keyValidationDto[Constants.APPLICATION_ID] + ":" + "/petstore/v1"
	return requestStreamDTO
}

func publishEvent(throttleEvent dtos.RequestStreamDTO) {
	log.Debugf(Constants.KEY_THROTTLE_FILTER + "Checking application sending throttle event to another worker.")
	PublishNonThrottleEvent(throttleEvent)
}
