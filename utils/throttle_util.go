package utils

import (
	"envoy-test-filter/Constants"
	"envoy-test-filter/dtos"
	"strings"
)

func GetPolicyDetails(deployedPolicies map[string]map[string]string, policyName string,
	prefix string) map[string]string {
	policyDetails := map[string]string{}
	if strings.EqualFold(policyName, Constants.UNAUTHENTICATED_TIER) || len(policyName) == 0 {
		policyDetails = map[string]string {
			"count": "1",
			"unitTime": "1",
			"timeUnit": "min",
			"stopOnQuota": "true",
		}
		return policyDetails
	}
	return policyDetails
}

func publishNonThrottleEvent(throttleEvent dtos.RequestStreamDTO) {
	//Publish throttle event to internal policies
	enabledGlobalTMEventPublishing := false
	if !enabledGlobalTMEventPublishing {
		publishNonThrottledEvent(throttleEvent)
	}
}
