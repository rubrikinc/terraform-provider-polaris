// Copyright 2025 Rubrik, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package provider

import (
	"reflect"
	"testing"

	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/aws"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/azure"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/core"
	"github.com/rubrikinc/rubrik-polaris-sdk-for-go/pkg/polaris/graphql/gcp"
)

func TestSortedAWSFeatures(t *testing.T) {
	input := []aws.FeaturePermissionGroups{
		{
			Feature: "EXOCOMPUTE",
			PermissionGroups: []aws.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupRSCManagedCluster},
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
		{
			Feature: "CLOUD_NATIVE_PROTECTION",
			PermissionGroups: []aws.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
	}

	result := sortedAWSFeatures(input)

	// Verify features are sorted by name.
	if result[0][keyName] != "CLOUD_NATIVE_PROTECTION" {
		t.Errorf("expected first feature to be CLOUD_NATIVE_PROTECTION, got %s", result[0][keyName])
	}
	if result[1][keyName] != "EXOCOMPUTE" {
		t.Errorf("expected second feature to be EXOCOMPUTE, got %s", result[1][keyName])
	}

	// Verify permission groups are sorted.
	exoGroups := result[1][keyPermissionGroups].([]string)
	expectedGroups := []string{"BASIC", "RSC_MANAGED_CLUSTER"}
	if !reflect.DeepEqual(exoGroups, expectedGroups) {
		t.Errorf("expected permission groups %v, got %v", expectedGroups, exoGroups)
	}
}

func TestSortedAzureFeatures(t *testing.T) {
	input := []azure.FeaturePermissionGroups{
		{
			Feature: "EXOCOMPUTE",
			PermissionGroups: []azure.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupRSCManagedCluster},
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
		{
			Feature: "CLOUD_NATIVE_PROTECTION",
			PermissionGroups: []azure.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
	}

	result := sortedAzureFeatures(input)

	// Verify features are sorted by name.
	if result[0][keyName] != "CLOUD_NATIVE_PROTECTION" {
		t.Errorf("expected first feature to be CLOUD_NATIVE_PROTECTION, got %s", result[0][keyName])
	}
	if result[1][keyName] != "EXOCOMPUTE" {
		t.Errorf("expected second feature to be EXOCOMPUTE, got %s", result[1][keyName])
	}

	// Verify permission groups are sorted.
	exoGroups := result[1][keyPermissionGroups].([]string)
	expectedGroups := []string{"BASIC", "RSC_MANAGED_CLUSTER"}
	if !reflect.DeepEqual(exoGroups, expectedGroups) {
		t.Errorf("expected permission groups %v, got %v", expectedGroups, exoGroups)
	}
}

func TestSortedGCPFeatures(t *testing.T) {
	input := []gcp.FeaturePermissionGroups{
		{
			Feature: "EXOCOMPUTE",
			PermissionGroups: []gcp.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupRSCManagedCluster},
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
		{
			Feature: "CLOUD_NATIVE_PROTECTION",
			PermissionGroups: []gcp.PermissionGroupInfo{
				{PermissionGroup: core.PermissionGroupBasic},
			},
		},
	}

	result := sortedGCPFeatures(input)

	// Verify features are sorted by name.
	if result[0][keyName] != "CLOUD_NATIVE_PROTECTION" {
		t.Errorf("expected first feature to be CLOUD_NATIVE_PROTECTION, got %s", result[0][keyName])
	}
	if result[1][keyName] != "EXOCOMPUTE" {
		t.Errorf("expected second feature to be EXOCOMPUTE, got %s", result[1][keyName])
	}

	// Verify permission groups are sorted.
	exoGroups := result[1][keyPermissionGroups].([]string)
	expectedGroups := []string{"BASIC", "RSC_MANAGED_CLUSTER"}
	if !reflect.DeepEqual(exoGroups, expectedGroups) {
		t.Errorf("expected permission groups %v, got %v", expectedGroups, exoGroups)
	}
}
