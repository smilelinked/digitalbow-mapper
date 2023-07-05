module github.com/smilelinkd/digitalbow-mapper

go 1.18

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/kubeedge/mappers-go v1.13.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.4
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.100.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/kubeedge/beehive v0.0.0 => github.com/kubeedge/beehive v1.7.0
	github.com/kubeedge/viaduct v0.0.0 => github.com/kubeedge/viaduct v0.0.0-20201130063818-e33931917980
)
