module github.com/myersguo/cc-mono/extensions/example

go 1.23

require (
	github.com/myersguo/cc-mono/pkg/agent v0.0.0
	github.com/myersguo/cc-mono/pkg/ai v0.0.0
	github.com/myersguo/cc-mono/pkg/shared v0.0.0
)

require golang.org/x/sync v0.10.0 // indirect

replace (
	github.com/myersguo/cc-mono/pkg/agent => ../../pkg/agent
	github.com/myersguo/cc-mono/pkg/ai => ../../pkg/ai
	github.com/myersguo/cc-mono/pkg/shared => ../../pkg/shared
)
