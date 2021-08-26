module gitlab.com/lightmeter/controlcenter/agent

go 1.16

require (
	github.com/bmatsuo/lmdb-go v1.8.0 // indirect
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/docker/docker v20.10.8+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/smartystreets/goconvey v1.6.4
	gitlab.com/lightmeter/controlcenter v0.0.0-20210630164959-79962c1334d8
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/grpc v1.40.0 // indirect
)

replace gitlab.com/lightmeter/controlcenter => ../
