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
//
// @contact  zampo3380@gmail.com

package interceptor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/go-anyway/framework-log"
	"github.com/go-anyway/framework-trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TraceUnaryInterceptor 创建一个 gRPC 一元拦截器，支持 OpenTelemetry
func TraceUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 从 metadata 中提取追踪信息
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			propagator := otel.GetTextMapPropagator()
			ctx = propagator.Extract(ctx, metadataCarrier(md))
		}

		// 开始新的 span
		ctx, span := trace.StartSpan(ctx, info.FullMethod)
		defer span.End()

		// 从 metadata 中提取 traceID 和 requestID
		var traceID, requestID string
		if ok && md != nil {
			if values := md.Get("x-trace-id"); len(values) > 0 {
				traceID = values[0]
			}
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}

		// 如果不存在，从 OpenTelemetry context 获取
		if traceID == "" {
			traceID = trace.TraceIDFromContext(ctx)
		}
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 注入到 context
		if traceID != "" {
			ctx = log.ContextWithTraceID(ctx, traceID)
		}
		if requestID != "" {
			ctx = log.ContextWithRequestID(ctx, requestID)
		}

		// 记录请求开始
		if traceID != "" || requestID != "" {
			logger := log.FromContext(ctx)
			logger.Info("gRPC request started",
				zap.String("method", info.FullMethod),
				zap.String("trace_id", traceID),
				zap.String("span_id", trace.SpanIDFromContext(ctx)),
			)
		}

		// 调用实际的处理器
		resp, err := handler(ctx, req)

		// 设置 span 属性
		span.SetAttributes(
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.status_code", status.Code(err).String()),
		)

		// 记录请求完成
		if traceID := log.TraceIDFromContext(ctx); traceID != "" || log.RequestIDFromContext(ctx) != "" {
			logger := log.FromContext(ctx)
			if err != nil {
				logger.Error("gRPC request failed",
					zap.String("method", info.FullMethod),
					zap.Error(err),
				)
			} else {
				logger.Info("gRPC request completed",
					zap.String("method", info.FullMethod),
				)
			}
		}

		return resp, err
	}
}

// metadataCarrier 实现 TextMapCarrier 接口（用于 OpenTelemetry 传播）
type metadataCarrier metadata.MD

func (m metadataCarrier) Get(key string) string {
	values := metadata.MD(m).Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

func (m metadataCarrier) Set(key, value string) {
	metadata.MD(m).Set(key, value)
}

func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TraceUnaryClientInterceptor 创建一个 gRPC 客户端一元拦截器，支持 OpenTelemetry
// 用于在客户端调用 gRPC 服务时注入追踪上下文并创建子 span
func TraceUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 开始新的 span（作为子 span）
		ctx, span := trace.StartSpan(ctx, method)
		defer span.End()

		// 从 context 中提取追踪信息并注入到 metadata
		propagator := otel.GetTextMapPropagator()
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		// 使用 OpenTelemetry 标准传播机制注入追踪上下文
		carrier := metadataCarrier(md)
		propagator.Inject(ctx, carrier)

		// 将 metadata 添加到 context
		ctx = metadata.NewOutgoingContext(ctx, md)

		// 设置 span 属性
		span.SetAttributes(
			attribute.String("rpc.method", method),
			attribute.String("rpc.system", "grpc"),
		)

		// 调用实际的 gRPC 方法
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 设置状态码
		if err != nil {
			span.SetAttributes(
				attribute.String("rpc.status_code", status.Code(err).String()),
			)
			span.RecordError(err)
		} else {
			span.SetAttributes(
				attribute.String("rpc.status_code", "OK"),
			)
		}

		return err
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用时间戳作为后备方案
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
