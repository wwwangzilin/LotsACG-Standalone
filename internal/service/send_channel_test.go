package service

import (
	"testing"

	"github.com/wwwangzilin/LotsACG-Standalone/internal/model/entity"
)

func mkArtwork(r18 bool, tags []string, artistName string) *entity.CachedArtworkData {
	return &entity.CachedArtworkData{
		Title: "test",
		R18:   r18,
		Tags:  tags,
		Artist: &entity.CachedArtist{
			Name: artistName,
		},
	}
}

func TestMatchSendChannelRule_Disabled(t *testing.T) {
	ch := &SendChannel{ChatID: 1, Enabled: false}
	if MatchSendChannelRule(ch, mkArtwork(false, nil, "a")) {
		t.Error("disabled channel should not match")
	}
}

func TestMatchSendChannelRule_R18(t *testing.T) {
	cases := []struct {
		mode string
		r18  bool
		want bool
	}{
		{SendChannelR18Follow, true, true},
		{SendChannelR18Follow, false, true},
		{SendChannelR18Allow, true, true},
		{SendChannelR18Allow, false, true},
		{SendChannelR18Deny, true, false},
		{SendChannelR18Deny, false, true},
		{SendChannelR18Only, true, true},
		{SendChannelR18Only, false, false},
	}
	for _, c := range cases {
		ch := &SendChannel{ChatID: 1, Enabled: true, R18Mode: c.mode}
		if got := MatchSendChannelRule(ch, mkArtwork(c.r18, nil, "a")); got != c.want {
			t.Errorf("mode=%s r18=%v got=%v want=%v", c.mode, c.r18, got, c.want)
		}
	}
}

func TestMatchSendChannelRule_Tags(t *testing.T) {
	art := mkArtwork(false, []string{"R-18", "girl", "blue"}, "artist")

	// include: 任一命中
	ch := &SendChannel{Enabled: true, IncludeTags: []string{"nope", "girl"}}
	if !MatchSendChannelRule(ch, art) {
		t.Error("include tag should match")
	}
	ch = &SendChannel{Enabled: true, IncludeTags: []string{"nope", "nope2"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("include tag mismatch should not match")
	}
	// include 大小写不敏感
	ch = &SendChannel{Enabled: true, IncludeTags: []string{"GIRL"}}
	if !MatchSendChannelRule(ch, art) {
		t.Error("include tag should be case-insensitive")
	}
	// exclude: 任一命中即拒绝
	ch = &SendChannel{Enabled: true, ExcludeTags: []string{"blue"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("exclude tag should reject")
	}
	ch = &SendChannel{Enabled: true, ExcludeTags: []string{"red", "BLUE"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("exclude tag (case-insensitive) should reject")
	}
	ch = &SendChannel{Enabled: true, ExcludeTags: []string{"red"}}
	if !MatchSendChannelRule(ch, art) {
		t.Error("non-matching exclude should pass")
	}
}

func TestMatchSendChannelRule_Artists(t *testing.T) {
	art := mkArtwork(false, nil, "Pixiv_Artist")

	ch := &SendChannel{Enabled: true, IncludeArtists: []string{"other", "pixiv_artist"}}
	if !MatchSendChannelRule(ch, art) {
		t.Error("include artist should match (case-insensitive)")
	}
	ch = &SendChannel{Enabled: true, IncludeArtists: []string{"other"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("include artist mismatch should not match")
	}
	ch = &SendChannel{Enabled: true, ExcludeArtists: []string{"pixiv_artist"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("exclude artist should reject")
	}
	ch = &SendChannel{Enabled: true, ExcludeArtists: []string{"other"}}
	if !MatchSendChannelRule(ch, art) {
		t.Error("non-matching exclude artist should pass")
	}
}

func TestMatchSendChannelRule_Combined(t *testing.T) {
	// deny R18 + include tag
	art := mkArtwork(true, []string{"R-18", "girl"}, "a")
	ch := &SendChannel{Enabled: true, R18Mode: SendChannelR18Deny, IncludeTags: []string{"girl"}}
	if MatchSendChannelRule(ch, art) {
		t.Error("R18 deny should reject R18 even with matching tag")
	}
	// only R18 + exclude artist
	art2 := mkArtwork(true, []string{"R-18"}, "banned")
	ch2 := &SendChannel{Enabled: true, R18Mode: SendChannelR18Only, ExcludeArtists: []string{"banned"}}
	if MatchSendChannelRule(ch2, art2) {
		t.Error("exclude artist should reject")
	}
}
