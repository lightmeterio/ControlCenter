package parser

func Fuzz(data []byte) int {
	_, _, err := Parse(data)

	if !IsRecoverableError(err) {
		return 0
	}

	return 1
}
