module github.com/go-anyway/framework-interceptor

go 1.25.4

require (
	github.com/go-anyway/framework-log v1.0.0
	github.com/go-anyway/framework-metrics v1.0.0
	github.com/go-anyway/framework-trace v1.0.0
	google.golang.org/grpc v1.78.0
)

replace (
	github.com/go-anyway/framework-log => ../core/log
	github.com/go-anyway/framework-metrics => ../metrics
	github.com/go-anyway/framework-trace => ../trace
)
