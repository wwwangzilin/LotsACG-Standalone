package handlers

import (
	"strings"
	"testing"
)

func TestParseChannelOpts(t *testing.T) {
	cases := []struct {
		args []string
		want *channelOpts
		rest []string
	}{
		{
			args: []string{"-r18=deny", "-link", "-t=a,b", "-x=c", "-at=x1", "-ax=y1", "id", "title"},
			want: &channelOpts{
				r18:            "deny",
				linkOnly:       boolPtr(true),
				includeTags:    []string{"a", "b"},
				excludeTags:    []string{"c"},
				includeArt:     []string{"x1"},
				excludeArt:     []string{"y1"},
				setIncludeTags: true,
				setExcludeTags: true,
				setIncludeArt:  true,
				setExcludeArt:  true,
			},
			rest: []string{"id", "title"},
		},
		{
			args: []string{"-nolink", "-title=My Channel"},
			want: &channelOpts{linkOnly: boolPtr(false), title: "My Channel"},
			rest: nil,
		},
		{
			args: []string{"-r18=only", "123"},
			want: &channelOpts{r18: "only"},
			rest: []string{"123"},
		},
		{
			args: []string{"-t=", "-x=   "},
			want: &channelOpts{
				includeTags:    nil,
				excludeTags:    nil,
				setIncludeTags: true,
				setExcludeTags: true,
			},
			rest: nil,
		},
	}
	for _, c := range cases {
		opts, rest, err := parseChannelOpts(c.args)
		if err != nil {
			t.Fatalf("parseChannelOpts(%v) err: %v", c.args, err)
		}
		if !channelOptsEqual(opts, c.want) {
			t.Errorf("parseChannelOpts(%v) = %+v, want %+v", c.args, opts, c.want)
		}
		if strings.Join(rest, " ") != strings.Join(c.rest, " ") {
			t.Errorf("rest = %v, want %v", rest, c.rest)
		}
	}
}

func TestParseChannelOpts_Invalid(t *testing.T) {
	if _, _, err := parseChannelOpts([]string{"-r18=bogus"}); err == nil {
		t.Error("invalid -r18 should error")
	}
	if _, _, err := parseChannelOpts([]string{"-unknown=1"}); err == nil {
		t.Error("unknown option should error")
	}
}

func TestSplitCommaList(t *testing.T) {
	if got := splitCommaList("a, b ,,c"); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("splitCommaList = %v", got)
	}
	if got := splitCommaList(""); got != nil {
		t.Errorf("empty should be nil, got %v", got)
	}
	if got := splitCommaList("  "); got != nil {
		t.Errorf("blank should be nil, got %v", got)
	}
}

func boolPtr(b bool) *bool { return &b }

func channelOptsEqual(a, b *channelOpts) bool {
	if (a.linkOnly == nil) != (b.linkOnly == nil) {
		return false
	}
	if a.linkOnly != nil && *a.linkOnly != *b.linkOnly {
		return false
	}
	if a.r18 != b.r18 || a.title != b.title {
		return false
	}
	if a.setIncludeTags != b.setIncludeTags || a.setExcludeTags != b.setExcludeTags ||
		a.setIncludeArt != b.setIncludeArt || a.setExcludeArt != b.setExcludeArt {
		return false
	}
	return strings.Join(a.includeTags, ",") == strings.Join(b.includeTags, ",") &&
		strings.Join(a.excludeTags, ",") == strings.Join(b.excludeTags, ",") &&
		strings.Join(a.includeArt, ",") == strings.Join(b.includeArt, ",") &&
		strings.Join(a.excludeArt, ",") == strings.Join(b.excludeArt, ",")
}
