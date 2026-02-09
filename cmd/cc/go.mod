module github.com/myersguo/cc-mono/cmd/cc

go 1.24.2

toolchain go1.24.11

replace (
	github.com/myersguo/cc-mono/internal/tui => ../../internal/tui
	github.com/myersguo/cc-mono/pkg/agent => ../../pkg/agent
	github.com/myersguo/cc-mono/pkg/ai => ../../pkg/ai
	github.com/myersguo/cc-mono/pkg/codingagent => ../../pkg/codingagent
	github.com/myersguo/cc-mono/pkg/rpc => ../../pkg/rpc
	github.com/myersguo/cc-mono/pkg/shared => ../../pkg/shared
)

require (
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/myersguo/cc-mono/extensions/example v0.0.0-20260206104409-47418203c36f
	github.com/myersguo/cc-mono/internal/tui v0.0.0-00010101000000-000000000000
	github.com/myersguo/cc-mono/pkg/agent v0.0.0
	github.com/myersguo/cc-mono/pkg/ai v0.0.0
	github.com/myersguo/cc-mono/pkg/codingagent v0.0.0-00010101000000-000000000000
	github.com/myersguo/cc-mono/pkg/rpc v0.0.0-00010101000000-000000000000
	github.com/myersguo/cc-mono/pkg/shared v0.0.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/bubbles v0.21.1 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.11.5 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/parsers/json v1.0.0 // indirect
	github.com/knadh/koanf/parsers/yaml v0.1.0 // indirect
	github.com/knadh/koanf/providers/rawbytes v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
