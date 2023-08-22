package parsing

import (
	"encoding/base64"
	"strings"

	"github.com/bitknox/hls-proxy/model"
)

func ParseInputUrl(inputString string) (*model.Input, error) {
	var out = &model.Input{}
	decodedBytes, err := base64.StdEncoding.DecodeString(inputString)
	parts := strings.Split(string(decodedBytes), "|")

	if len(parts) == 2 {
		out.Url = parts[0]
		out.FileType = parts[1]
		return out, nil
	}

	if len(parts) == 3 {
		out.Url = parts[0]
		out.Referer = parts[1]
		out.FileType = parts[2]
		return out, nil
	}

	if len(parts) == 4 {
		out.Url = parts[0]
		out.Referer = parts[1]
		out.Origin = parts[2]
		out.FileType = parts[3]
		return out, nil
	}

	if err != nil {
		return out, nil
	}

	return out, nil
}
