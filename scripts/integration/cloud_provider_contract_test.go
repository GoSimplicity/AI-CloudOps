package main

import (
	"strings"
	"testing"
)

const matchingProviders = `export enum CloudProvider {
  AliCloud = 1,
  TencentCloud = 2,
  AWS = 3,
  HuaweiCloud = 4,
  Azure = 5,
  GCP = 6,
}`

func TestCheckCloudProviders(t *testing.T) {
	for name, source := range map[string]string{
		"correct":         matchingProviders,
		"different value": strings.Replace(matchingProviders, "AWS = 3", "AWS = 4", 1),
		"missing":         strings.Replace(matchingProviders, "  GCP = 6,\n", "", 1),
		"duplicate":       strings.Replace(matchingProviders, "  GCP = 6,", "  GCP = 6,\n  GCP = 6,", 1),
		"unknown":         strings.Replace(matchingProviders, "  GCP = 6,", "  GCP = 6,\n  Other = 7,", 1),
		"no enum":         "export enum Other {}",
	} {
		t.Run(name, func(t *testing.T) {
			err := checkCloudProviders(source)
			if (err == nil) != (name == "correct") {
				t.Fatalf("checkCloudProviders() error = %v", err)
			}
		})
	}
}
