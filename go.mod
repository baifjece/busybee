module github.com/deepfabric/busybee

go 1.13

require (
	github.com/RoaringBitmap/roaring v0.4.21
	github.com/buger/jsonparser v0.0.0-20191204142016-1a29609e0929
	github.com/deepfabric/beehive v0.0.0-20200205103359-796376e893cd
	github.com/deepfabric/prophet v0.0.0-20191202055442-cec7351ee5cf
	github.com/fagongzi/expr v0.0.0-20200111083631-9072dbb0af1c
	github.com/fagongzi/goetty v1.3.4
	github.com/fagongzi/log v0.0.0-20191122063922-293b75312445
	github.com/fagongzi/util v0.0.0-20200205003627-8cf7ebc854c9
	github.com/gogo/protobuf v1.3.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/shirou/gopsutil v2.19.10+incompatible // indirect
	github.com/stretchr/testify v1.4.0
	golang.org/x/text v0.3.2 // indirect
)

replace github.com/coreos/etcd => github.com/deepfabric/etcd v3.3.17+incompatible
