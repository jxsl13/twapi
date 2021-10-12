package config

func skipWhitespace(data []byte) (index int) {
	if len(data) == 0 {
		return -1
	}
	resultIndex := 0

	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
		resultIndex++
	}
	return resultIndex
}

func skipToWhitespace(data []byte) (index int) {
	if len(data) == 0 {
		return -1
	}
	resultIndex := 0
	for _, b := range data {
		if b == ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
		resultIndex++
	}
	return resultIndex
}
