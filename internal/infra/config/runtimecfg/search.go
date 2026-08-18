package runtimecfg

import "errors"

type SearchConfig struct {
	MeiliSearch    MeiliSearchConfig `toml:"meilisearch" mapstructure:"meilisearch" json:"meilisearch" yaml:"meilisearch"`
	Engine         string            `toml:"engine" mapstructure:"engine" json:"engine" yaml:"engine"`
	Enable         bool              `toml:"enable" mapstructure:"enable" json:"enable" yaml:"enable"`
	OrbMinMatches  int               `toml:"orb_min_matches" mapstructure:"orb_min_matches" json:"orb_min_matches" yaml:"orb_min_matches"`
	OrbMinScore    float64           `toml:"orb_min_score" mapstructure:"orb_min_score" json:"orb_min_score" yaml:"orb_min_score"`
	DupCheckEnable bool              `toml:"dup_check_enable" mapstructure:"dup_check_enable" json:"dup_check_enable" yaml:"dup_check_enable"`
}

type MeiliSearchConfig struct {
	Host     string `toml:"host" mapstructure:"host" json:"host" yaml:"host"`
	Key      string `toml:"key" mapstructure:"key" json:"key" yaml:"key"`
	Index    string `toml:"index" mapstructure:"index" json:"index" yaml:"index"`
	Embedder string `toml:"embedder" mapstructure:"embedder" json:"embedder" yaml:"embedder"`
}

func (c MeiliSearchConfig) Valid() error {
	if c.Host == "" {
		return errors.New("meilisearch host is empty")
	}
	if c.Index == "" {
		return errors.New("meilisearch index is empty")
	}
	return nil
}
