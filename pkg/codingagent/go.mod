module github.com/myersguo/cc-mono/pkg/codingagent

go 1.23.0

toolchain go1.24.11

require (
	github.com/fsnotify/fsnotify v1.8.0
	github.com/knadh/koanf/parsers/json v1.0.0
	github.com/knadh/koanf/parsers/yaml v0.1.0
	github.com/knadh/koanf/providers/rawbytes v1.0.0
	github.com/knadh/koanf/v2 v2.1.2
)

require (
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	golang.org/x/sys v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/myersguo/cc-mono/pkg/agent => ../agent
	github.com/myersguo/cc-mono/pkg/ai => ../ai
	github.com/myersguo/cc-mono/pkg/shared => ../shared
)
