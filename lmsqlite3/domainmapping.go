package lmsqlite3

import (
	"gitlab.com/lightmeter/controlcenter/domainmapping"
)

func resolveDomainMapping(domain string) string {
	return domainmapping.Resolve(domain)
}
