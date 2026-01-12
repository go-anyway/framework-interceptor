// Copyright 2025 zampo.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interceptor

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestMetadataCarrier_Get(t *testing.T) {
	md := metadata.New(map[string]string{
		"x-test-key":  "test-value",
		"x-another":   "another-value",
		"x-empty-key": "",
	})

	carrier := metadataCarrier(md)

	tests := []struct {
		name   string
		key    string
		want   string
		wantOK bool
	}{
		{
			name:   "existing key",
			key:    "x-test-key",
			want:   "test-value",
			wantOK: true,
		},
		{
			name:   "another existing key",
			key:    "x-another",
			want:   "another-value",
			wantOK: true,
		},
		{
			name:   "non-existing key",
			key:    "x-not-exist",
			want:   "",
			wantOK: false,
		},
		{
			name:   "empty value key",
			key:    "x-empty-key",
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := carrier.Get(tt.key)
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestMetadataCarrier_Set(t *testing.T) {
	md := metadata.New(map[string]string{})
	carrier := metadataCarrier(md)

	carrier.Set("x-new-key", "new-value")

	got := carrier.Get("x-new-key")
	if got != "new-value" {
		t.Errorf("Set() followed by Get() = %q, want %q", got, "new-value")
	}
}

func TestMetadataCarrier_Keys(t *testing.T) {
	md := metadata.New(map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	})

	carrier := metadataCarrier(md)
	keys := carrier.Keys()

	if len(keys) != 3 {
		t.Errorf("Keys() returned %d keys, want 3", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	expectedKeys := []string{"key1", "key2", "key3"}
	for _, expected := range expectedKeys {
		if !keySet[expected] {
			t.Errorf("Keys() missing expected key %q", expected)
		}
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("generateRequestID() returned empty string")
	}

	if len(id1) != 32 {
		t.Errorf("generateRequestID() returned string of length %d, want 32", len(id1))
	}

	if id1 == id2 {
		t.Error("generateRequestID() returned same ID for two calls")
	}
}

func TestTraceUnaryInterceptor_ReturnsFunction(t *testing.T) {
	interceptor := TraceUnaryInterceptor()

	if interceptor == nil {
		t.Error("TraceUnaryInterceptor() returned nil")
	}

	if interceptor != nil {
		_ = interceptor
	}
}

func TestMetricsUnaryInterceptor_ReturnsFunction(t *testing.T) {
	interceptor := MetricsUnaryInterceptor()

	if interceptor == nil {
		t.Error("MetricsUnaryInterceptor() returned nil")
	}

	if interceptor != nil {
		_ = interceptor
	}
}

func TestTraceUnaryInterceptor_HandlerInvocation(t *testing.T) {
	interceptor := TraceUnaryInterceptor()

	var handlerCalled bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor(ctx, nil, info, handler)

	if err != nil {
		t.Errorf("interceptor() returned unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}
}

func TestMetricsUnaryInterceptor_HandlerInvocation(t *testing.T) {
	interceptor := MetricsUnaryInterceptor()

	var handlerCalled bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor(ctx, nil, info, handler)

	if err != nil {
		t.Errorf("interceptor() returned unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("handler was not called")
	}
}

func TestTraceUnaryInterceptor_WithError(t *testing.T) {
	interceptor := TraceUnaryInterceptor()

	expectedErr := context.DeadlineExceeded
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor(ctx, nil, info, handler)

	if !errors.Is(err, expectedErr) {
		t.Errorf("interceptor() returned error %v, want %v", err, expectedErr)
	}
}

func TestMetricsUnaryInterceptor_WithError(t *testing.T) {
	interceptor := MetricsUnaryInterceptor()

	expectedErr := context.DeadlineExceeded
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	_, err := interceptor(ctx, nil, info, handler)

	if !errors.Is(err, expectedErr) {
		t.Errorf("interceptor() returned error %v, want %v", err, expectedErr)
	}
}
