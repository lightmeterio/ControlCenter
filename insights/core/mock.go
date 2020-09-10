// +build !release

//go:generate go run github.com/golang/mock/mockgen -destination=mock/insights_mock.go gitlab.com/lightmeter/controlcenter/insights/core Fetcher

package core
