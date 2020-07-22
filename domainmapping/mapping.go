package domainmapping

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util"
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
	r, ok := m.r[domain]

	if !ok {
		return domain
	}

	return r
}

var DefaultMapping *Mapper
