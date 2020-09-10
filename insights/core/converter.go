package core

import (
	"errors"
	"log"
)

var (
	ErrInvalidContentType = errors.New("Unknown Insight Content Type")
)

type decoderHandler func([]byte) (interface{}, error)

type converterHandler struct {
	value   int
	decoder decoderHandler
}

var (
	converters        = map[string]converterHandler{}
	reverseConverters = map[int]string{}
)

func ContentTypeForValue(value int) (string, error) {
	c, ok := reverseConverters[value]

	if !ok {
		return "", ErrInvalidContentType
	}

	return c, nil
}

func ValueForContentType(contentType string) (int, error) {
	v, ok := converters[contentType]

	if !ok {
		return 0, ErrInvalidContentType
	}

	return v.value, nil
}

func RegisterContentType(contentType string, value int, handler decoderHandler) {
	c, ok := reverseConverters[value]

	if ok {
		log.Panicln("A content converter with value", value, "is already registred:", c, ". You must use a different and unique number!")
		return
	}

	converters[contentType] = converterHandler{
		value:   value,
		decoder: handler,
	}

	reverseConverters[value] = contentType
}

func decodeByContentType(contentType string, content []byte) (interface{}, error) {
	v, ok := converters[contentType]

	if !ok {
		return nil, ErrInvalidContentType
	}

	return v.decoder(content)
}
