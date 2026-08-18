package mediatool

import (
	"image"
	"image/color"
	"testing"
)

func TestGetImageORBFeatures(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}

	encoded, err := GetImageORBFeatures(img)
	if err != nil {
		t.Fatalf("GetImageORBFeatures returned error: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty ORB features")
	}

	matches, score, err := MatchORBFeatures(encoded, encoded, 1, 0.1)
	if err != nil {
		t.Fatalf("MatchORBFeatures returned error: %v", err)
	}
	if matches <= 0 {
		t.Fatal("expected at least one match for identical images")
	}
	if score <= 0 {
		t.Fatal("expected positive match score for identical images")
	}
}

func TestMatchORBFeaturesOnDifferentImages(t *testing.T) {
	left := image.NewGray(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			left.SetGray(x, y, color.Gray{Y: uint8((x + y) % 256)})
		}
	}
	right := image.NewGray(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			right.SetGray(x, y, color.Gray{Y: uint8((x*2 + y*3) % 256)})
		}
	}

	leftEncoded, err := GetImageORBFeatures(left)
	if err != nil {
		t.Fatalf("GetImageORBFeatures left returned error: %v", err)
	}
	rightEncoded, err := GetImageORBFeatures(right)
	if err != nil {
		t.Fatalf("GetImageORBFeatures right returned error: %v", err)
	}

	matches, score, err := MatchORBFeatures(leftEncoded, rightEncoded, 1, 0.1)
	if err != nil {
		t.Fatalf("MatchORBFeatures returned error: %v", err)
	}
	if matches < 0 {
		t.Fatal("expected a non-negative match count")
	}
	if score < 0 {
		t.Fatal("expected a non-negative match score")
	}
}
