package parser

func Fuzz(data []byte) int {
	_, _, err := Parse(data)

	if err != nil {
		return 0
	}

	return 1
}
