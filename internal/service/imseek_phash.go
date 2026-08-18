package service

import (
	"bytes"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/pkg/mediatool"
)

func getPhashFromBytes(imageBytes []byte) (string, error) {
	return mediatool.GetImagePhashFromReader(bytes.NewReader(imageBytes))
}
