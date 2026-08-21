package twitter

import (
	"reflect"
	"testing"
)

func TestBuildTwitterImageCandidates_WithProxy(t *testing.T) {
	got := BuildTwitterImageCandidates(
		"https://pbs.twimg.com/media/ABC123?format=jpg&name=orig",
		[]string{"twimg.example.com", "img2.example.org"},
	)
	want := []string{
		"https://twimg.example.com/media/ABC123?format=jpg&name=orig",
		"https://img2.example.org/media/ABC123?format=jpg&name=orig",
		"https://pbs.twimg.com/media/ABC123?format=jpg&name=orig",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildTwitterImageCandidates_NoProxy(t *testing.T) {
	got := BuildTwitterImageCandidates(
		"https://pbs.twimg.com/media/ABC123?format=jpg&name=orig",
		nil,
	)
	want := []string{"https://pbs.twimg.com/media/ABC123?format=jpg&name=orig"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildTwitterImageCandidates_NonTwimg(t *testing.T) {
	// 非 pbs.twimg.com 域名不做反代重写
	got := BuildTwitterImageCandidates(
		"https://example.com/img/1.png",
		[]string{"twimg.example.com"},
	)
	want := []string{"https://example.com/img/1.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestBuildTwitterImageCandidates_Dedupe(t *testing.T) {
	got := BuildTwitterImageCandidates(
		"https://pbs.twimg.com/media/ABC?format=jpg",
		[]string{"twimg.example.com", "twimg.example.com", ""},
	)
	want := []string{
		"https://twimg.example.com/media/ABC?format=jpg",
		"https://pbs.twimg.com/media/ABC?format=jpg",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestIsTwimgHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"pbs.twimg.com", true},
		{"video.twimg.com", false}, // 视频域名不走图片反代
		{"pbs.twimg.com.evil.com", false},
		{"example.com", false},
	}
	for _, c := range cases {
		if got := isTwimgHost(c.host); got != c.want {
			t.Errorf("isTwimgHost(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}
