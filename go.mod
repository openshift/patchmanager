module github.com/mfojtik/patchmanager

go 1.15

require (
	github.com/cheggaaa/pb/v3 v3.0.6
	github.com/enriquebris/goconcurrentcounter v0.0.0-20210303202617-b0fccc15d4ea // indirect
	github.com/enriquebris/goconcurrentqueue v0.6.0 // indirect
	github.com/enriquebris/goworkerpool v0.10.0
	github.com/eparis/bugzilla v0.0.0-20210108140723-998a521ca0b5
	github.com/google/go-github/v32 v32.1.0
	github.com/openshift/build-machinery-go v0.0.0-20210209125900-0da259a2c359
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/component-base v0.20.4
	k8s.io/klog/v2 v2.4.0
)

replace gopkg.in/yaml.v2 => github.com/DirectXMan12/go-yaml v0.0.0-20151006211019-4c95efea8631
