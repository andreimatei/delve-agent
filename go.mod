module github.com/andreimatei/delve-agent

go 1.19

require (
	github.com/go-delve/delve v1.20.2
	github.com/kr/pretty v0.2.1
)

require (
	github.com/cilium/ebpf v0.7.0 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/sirupsen/logrus v1.6.0 // indirect
	go.starlark.net v0.0.0-20220816155156-cfacd8902214 // indirect
	golang.org/x/arch v0.0.0-20190927153633-4e8777c89be4 // indirect
	golang.org/x/sys v0.0.0-20220908164124-27713097b956 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-delve/delve => ../../go-delve/delve
