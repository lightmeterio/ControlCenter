// +build !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/dashboard_mock.go gitlab.com/lightmeter/controlcenter/dashboard Dashboard

package dashboard
