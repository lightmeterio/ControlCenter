package domainmapping

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type RawList map[string][]string

type Mapper struct {
	l RawList
	r map[string]string
}

func invertMapping(l RawList) (map[string]string, error) {
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
	r, err := invertMapping(list)

	if err != nil {
		return Mapper{}, errorutil.Wrap(err)
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

func (m *Mapper) ForEach(f func(string, string) error) error {
	for k, v := range m.r {
		if err := f(k, v); err != nil {
			return errorutil.Wrap(err)
		}
	}

	return nil
}

var DefaultMapping Mapper
