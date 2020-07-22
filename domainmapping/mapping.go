package domainmapping

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util"
	"strings"
)

type RawList map[string][]string

type Mapper struct {
	l RawList
	r map[string]string
}

func reverseList(l RawList) (map[string]string, error) {
	result := make(map[string]string, len(l))

	for k, v := range l {
		for _, d := range v {
			if dr, repeated := result[d]; repeated {
				return nil, fmt.Errorf("Domain %s already mapped to %s", d, dr)
			}

			result[d] = k
		}
	}

	return result, nil
}

func Mapping(list RawList) (Mapper, error) {
	r, err := reverseList(list)

	if err != nil {
		return Mapper{}, util.WrapError(err)
	}

	return Mapper{l: list, r: r}, nil
}

func (m *Mapper) Resolve(domain string) string {
	lower := strings.ToLower(domain)

	r, ok := m.r[lower]

	if !ok {
		return lower
	}

	return r
}

func Resolve(domain string) string {
	return globalMapping.Resolve(domain)
}

var (
	globalMapping  *Mapper
	DefaultMapping *Mapper
)

func init() {
	m, _ := Mapping(RawList{})
	globalMapping = &m
}

func RegisterMapping(m *Mapper) {
	globalMapping = m
}

func RegisterDefaultMapping() {
	RegisterMapping(DefaultMapping)
}
