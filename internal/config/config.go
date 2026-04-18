// Package config handles loading the optional .columnar.json configuration
// file. All fields have sensible defaults so the CLI works out-of-the-box with
// no config present.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type LanguageOverride struct {
	CommentToken    string   `json:"commentToken"`
	AlignTokens     []string `json:"alignTokens"`
	ChainAlignStyle string   `json:"chainAlignStyle"`
}

type Config struct {
	MinColumnGap        int                         `json:"minColumnGap"`
	MaxColumnWidth      int                         `json:"maxColumnWidth"`
	IndentSize          int                         `json:"indentSize"`
	AlignAssignments    bool                        `json:"alignAssignments"`
	AlignOperators      bool                        `json:"alignOperators"`
	AlignComments       bool                        `json:"alignComments"`
	AlignMethodChains   bool                        `json:"alignMethodChains"`
	AlignTernary        bool                        `json:"alignTernary"`
	AlignEnums          bool                        `json:"alignEnums"`
	AlignSwitchCases    bool                        `json:"alignSwitchCases"`
	AlignMapEntries     bool                        `json:"alignMapEntries"`
	AlignStructFields   bool                        `json:"alignStructFields"`
	AlignImports        bool                        `json:"alignImports"`
	AlignFunctionParams bool                        `json:"alignFunctionParams"`
	AlignArrayColumns   bool                        `json:"alignArrayColumns"`
	Languages           map[string]LanguageOverride `json:"languages"`
}

// Default returns a config with every alignment feature enabled and sane
// defaults for gap/width.
func Default() *Config {
	return &Config{
		MinColumnGap:        1,
		MaxColumnWidth:      80,
		IndentSize:          4,
		AlignAssignments:    true,
		AlignOperators:      true,
		AlignComments:       true,
		AlignMethodChains:   true,
		AlignTernary:        true,
		AlignEnums:          true,
		AlignSwitchCases:    true,
		AlignMapEntries:     true,
		AlignStructFields:   true,
		AlignImports:        true,
		AlignFunctionParams: true,
		AlignArrayColumns:   true,
		Languages:           map[string]LanguageOverride{},
	}
}

// Load reads and parses a .columnar.json file, merging it on top of defaults.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	cfg := Default()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Languages == nil {
		cfg.Languages = map[string]LanguageOverride{}
	}
	if cfg.MinColumnGap <= 0 {
		cfg.MinColumnGap = 1
	}
	return cfg, nil
}
