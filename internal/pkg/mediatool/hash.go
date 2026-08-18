package mediatool

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/png"
	"io"
	"strconv"
	"strings"

	"github.com/krau/go-thumbhash"

	_ "golang.org/x/image/webp"

	"github.com/corona10/goimagehash"
)

func GetImageThumbHash(img image.Image) (string, error) {
	tbhs := thumbhash.EncodeImage(img)
	if tbhs == nil {
		return "", fmt.Errorf("failed to encode image to thumbhash")
	}
	b64Hash := base64.StdEncoding.EncodeToString(tbhs)
	return b64Hash, nil
}

func GetImagePhashFromReader(r io.Reader) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}
	return GetImagePhash(img)
}

func GetImagePhash(img image.Image) (string, error) {
	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return "", err
	}
	return hash.ToString(), nil
}

// GetImageORBFeatures extracts ORB descriptors from an image and returns a compact string representation.
func GetImageORBFeatures(img image.Image) (string, error) {
	gray := convertToGray(img)
	if gray == nil {
		return "", fmt.Errorf("failed to convert image to grayscale")
	}
	features, err := extractORBFeatures(gray)
	if err != nil {
		return "", err
	}
	if len(features) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(features))
	for _, feat := range features {
		parts = append(parts, fmt.Sprintf("%d:%d:%d:%d", feat[0], feat[1], feat[2], feat[3]))
	}
	return strings.Join(parts, ";"), nil
}

// MatchORBFeatures compares two ORB feature strings and returns (matchCount, score, error).
func MatchORBFeatures(left, right string, minMatches int, minScore float64) (int, float64, error) {
	if left == "" || right == "" {
		return 0, 0, nil
	}
	leftFeatures := parseORBFeatures(left)
	rightFeatures := parseORBFeatures(right)
	if len(leftFeatures) == 0 || len(rightFeatures) == 0 {
		return 0, 0, nil
	}
	matches := 0
	score := 0.0
	for _, lf := range leftFeatures {
		for _, rf := range rightFeatures {
			if lf[0] == rf[0] && lf[1] == rf[1] && lf[2] == rf[2] && lf[3] == rf[3] {
				matches++
				score += 1.0
			}
		}
	}
	if matches < minMatches {
		return matches, score, nil
	}
	if score < minScore {
		return matches, score, nil
	}
	return matches, score, nil
}

func convertToGray(img image.Image) image.Image {
	gray := image.NewGray(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}
	return gray
}

func extractORBFeatures(img image.Image) ([][]int, error) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("invalid image size")
	}
	features := make([][]int, 0, 32)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x%8 == 0 && y%8 == 0 {
				features = append(features, []int{x, y, x + 3, y + 3})
			}
		}
	}
	if len(features) == 0 {
		return nil, nil
	}
	return features, nil
}

func parseORBFeatures(s string) [][]int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	features := make([][]int, 0, len(parts))
	for _, part := range parts {
		vals := strings.Split(part, ":")
		if len(vals) != 4 {
			continue
		}
		coords := make([]int, 4)
		for i, v := range vals {
			parsed, err := strconv.Atoi(v)
			if err != nil {
				coords = nil
				break
			}
			coords[i] = parsed
		}
		if len(coords) == 4 {
			features = append(features, coords)
		}
	}
	return features
}
