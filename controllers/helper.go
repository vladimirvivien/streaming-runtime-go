package controllers

import (
	"fmt"
	"strings"
)

func getMetadataStringValues(keyName string, properties map[string]string) (result []string) {
	for key, value := range properties {
		if key == keyName {
			result = append(result, value)
		}
	}
	return
}

// validateTarget returns component/route if ok, or component/component if route is missing.
func validateTarget(target string) string {
	if target == "" {
		return target
	}
	if strings.Index(target, "/") == -1 {
		return fmt.Sprintf("%s/%s", target, target)
	}
	return target
}
