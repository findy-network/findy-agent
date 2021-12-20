package utils

import "encoding/base64"

func DecodeB64(str string) ([]byte, error) {
	data, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		data, err = base64.RawURLEncoding.DecodeString(str)
	}
	return data, err
}
