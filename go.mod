module github.com/projecteru2/yavirt

go 1.16

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/containernetworking/cni v0.8.1
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/getsentry/sentry-go v0.9.0
	github.com/gin-gonic/gin v1.6.3
	github.com/google/uuid v1.2.0
	github.com/gophercloud/gophercloud v0.0.0-20190126172459-c818fa66e4c8 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20201030020617-7189b3728523 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/libvirt/libvirt-go v5.2.0+incompatible
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/projectcalico/api v0.0.0-20211207142834-757e73ac95b9
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml v0.0.0-20161201183616-955bc3e451ef // indirect
	github.com/projectcalico/libcalico-go v1.7.2-0.20211201231514-3402eca9d274
	github.com/projecteru2/core v0.0.0-20210317082513-84f470562415
	github.com/projecteru2/libyavirt v0.0.0-20220112061300-ac7002c411ff
	github.com/prometheus/client_golang v1.7.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/urfave/cli/v2 v2.3.0
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20210104183010-2eb08e3e575f // indirect
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b
	golang.org/x/sys v0.0.0-20210917161153-d61c044b1678
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.40.0
	gopkg.in/go-playground/validator.v9 v9.29.1 // indirect
	k8s.io/apimachinery v0.21.0
	k8s.io/klog v0.3.1 // indirect
)

replace (
	github.com/projectcalico/libcalico-go => github.com/projectcalico/calico v1.7.1-libcalico-go.0.20211201231514-3402eca9d274
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
)
