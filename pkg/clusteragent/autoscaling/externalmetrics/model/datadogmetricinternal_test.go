// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver

package model

import (
	"errors"
	"testing"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datadoghq "github.com/DataDog/datadog-operator/api/datadoghq/v1alpha1"

	"github.com/stretchr/testify/assert"
)

var (
	simpleQuery           = "avg:nginx.net.request_per_s{kube_container_name:nginx}"
	simpleQueryWithRollup = "avg:nginx.net.request_per_s{kube_container_name:nginx}.rollup(60)"
	templatedQuery        = "avg:nginx.net.request_per_s{kube_container_name:nginx,kube_cluster_name:%%tag_kube_cluster_name%%}"
	invalidTemplatedQuery = "avg:nginx.net.request_per_s{kube_container_name:nginx,kube_cluster_name:%%tag_foo%%}"
	resolvedQuery         = "avg:nginx.net.request_per_s{kube_container_name:nginx,kube_cluster_name:cluster-foo}"
)

func TestDatadogMetricInternal_UpdateFrom(t *testing.T) {
	templatedTags = templatedTagsStub
	currentTime := time.Now()
	tests := []struct {
		name                  string
		ddmInternal           *DatadogMetricInternal
		new                   datadoghq.DatadogMetric
		expectedQuery         string
		expectedResolvedQuery *string
		expectedTimewindow    time.Duration
		expectedMaxAge        time.Duration
		expectedAlwaysActive  bool
		expectedRetries       int
		expectedRetryAfter    time.Time
	}{
		{
			name: "same query",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "same query - annotation on",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						alwaysActiveAnnotation: "true",
					},
				},
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  true,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "same query - annotation off",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						alwaysActiveAnnotation: "false",
					},
				},
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "new query, no templating",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQueryWithRollup,
				},
			},
			expectedQuery:         simpleQueryWithRollup,
			expectedResolvedQuery: &simpleQueryWithRollup,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
		{
			name: "same query, nil ResolvedQuery",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: nil,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "new templated query",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: templatedQuery,
				},
			},
			expectedQuery:         templatedQuery,
			expectedResolvedQuery: &resolvedQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "cannot resolve query",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: invalidTemplatedQuery,
				},
			},
			expectedQuery:         invalidTemplatedQuery,
			expectedResolvedQuery: nil,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
		{
			name: "new max age",
			ddmInternal: &DatadogMetricInternal{
				MaxAge:        5 * time.Second,
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					MaxAge: v1.Duration{Duration: 10 * time.Second},
					Query:  simpleQuery,
				},
			},
			expectedMaxAge:        10 * time.Second,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
		{
			name: "same max age",
			ddmInternal: &DatadogMetricInternal{
				MaxAge:        5 * time.Second,
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					MaxAge: v1.Duration{Duration: 5 * time.Second},
					Query:  simpleQuery,
				},
			},
			expectedMaxAge:        5 * time.Second,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "deleted max age",
			ddmInternal: &DatadogMetricInternal{
				MaxAge:        5 * time.Second,
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedMaxAge:        0,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
		{
			name: "new time window",
			ddmInternal: &DatadogMetricInternal{
				TimeWindow:    time.Duration(5 * time.Second),
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					TimeWindow: v1.Duration{Duration: 10 * time.Second},
					Query:      simpleQuery,
				},
			},
			expectedTimewindow:    10 * time.Second,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
		{
			name: "same time window",
			ddmInternal: &DatadogMetricInternal{
				TimeWindow:    time.Duration(5 * time.Second),
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					TimeWindow: v1.Duration{Duration: 5 * time.Second},
					Query:      simpleQuery,
				},
			},
			expectedTimewindow:    5 * time.Second,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       1,
			expectedRetryAfter:    currentTime,
		},
		{
			name: "deleted time window",
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
				Retries:       1,
				RetryAfter:    currentTime,
			},
			new: datadoghq.DatadogMetric{
				Spec: datadoghq.DatadogMetricSpec{
					Query: simpleQuery,
				},
			},
			expectedTimewindow:    0,
			expectedQuery:         simpleQuery,
			expectedResolvedQuery: &simpleQuery,
			expectedAlwaysActive:  false,
			expectedRetries:       0,
			expectedRetryAfter:    time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ddmInternal.UpdateFrom(tt.new)
			assert.Equal(t, tt.expectedQuery, tt.ddmInternal.query)
			if tt.expectedResolvedQuery == nil {
				assert.Nil(t, tt.ddmInternal.resolvedQuery)
			} else {
				assert.Equal(t, *tt.expectedResolvedQuery, *tt.ddmInternal.resolvedQuery)
			}

			assert.Equal(t, tt.expectedMaxAge, tt.ddmInternal.MaxAge)
			assert.Equal(t, tt.expectedAlwaysActive, tt.ddmInternal.AlwaysActive)
		})
	}
}

func TestDatadogMetricInternal_resolveQuery(t *testing.T) {
	templatedTags = templatedTagsStub
	tests := []struct {
		name        string
		query       string
		ddmInternal *DatadogMetricInternal
		expected    *DatadogMetricInternal
	}{
		{
			name:  "simple query",
			query: simpleQuery,
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: nil,
			},
			expected: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
			},
		},
		{
			name:  "same templated query",
			query: templatedQuery,
			ddmInternal: &DatadogMetricInternal{
				query:         templatedQuery,
				resolvedQuery: nil,
			},
			expected: &DatadogMetricInternal{
				query:         templatedQuery,
				resolvedQuery: &resolvedQuery,
			},
		},
		{
			name:  "new templated query",
			query: templatedQuery,
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
			},
			expected: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &resolvedQuery,
			},
		},
		{
			name:  "invalid templated query",
			query: invalidTemplatedQuery,
			ddmInternal: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: &simpleQuery,
			},
			expected: &DatadogMetricInternal{
				query:         simpleQuery,
				resolvedQuery: nil,
				Valid:         false,
				Error:         errors.New(`Cannot resolve query: cannot resolve tag template "foo": tag is not supported`),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ddmInternal.resolveQuery(tt.query)
			assert.Equal(t, tt.expected.query, tt.ddmInternal.query)
			assert.Equal(t, tt.expected.resolvedQuery, tt.ddmInternal.resolvedQuery)
			assert.Equal(t, tt.expected.Valid, tt.ddmInternal.Valid)
			assert.Equal(t, tt.expected.Error, tt.ddmInternal.Error)
			if tt.expected.Error != nil {
				assert.NotEqual(t, tt.expected.UpdateTime, tt.ddmInternal.UpdateTime)
			}
		})
	}
}
