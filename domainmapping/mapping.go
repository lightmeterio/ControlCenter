package domainmapping

type RawList map[string][]string

type Mapper struct {
	l RawList
	r map[string]string
}

func reverseList(l RawList) map[string]string {
	result := make(map[string]string, len(l))

	for k, v := range l {
		for _, d := range v {
			result[d] = k
		}
	}

	return result
}

func Mapping(list RawList) *Mapper {
	return &Mapper{l: list, r: reverseList(list)}
}

func (m *Mapper) Resolve(domain string) string {
	r, ok := m.r[domain]

	if !ok {
		return domain
	}

	return r
}
