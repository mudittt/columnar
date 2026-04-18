package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mudittt/columnar/internal/config"
	"github.com/mudittt/columnar/internal/formatter"
	"github.com/spf13/cobra"
)

var (
	writeInPlace bool
	languageFlag string
	configPath   string
	diffOutput   bool
)

func init() {
	formatCmd.Flags().BoolVarP(&writeInPlace, "write", "w", false, "write result to file instead of stdout")
	formatCmd.Flags().StringVarP(&languageFlag, "language", "l", "", "explicitly set language (overrides extension detection)")
	formatCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to .columnar.json config file")
	formatCmd.Flags().BoolVarP(&diffOutput, "check", "d", false, "exit with non-zero if any file would be changed")
	rootCmd.AddCommand(formatCmd)
}

var formatCmd = &cobra.Command{
	Use:   "format [files...]",
	Short: "Format files with elastic tabstop alignment",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(configPath)
		if err != nil {
			return err
		}
		anyDirty := false
		for _, path := range args {
			changed, err := formatFile(path, cfg)
			if err != nil {
				return err
			}
			if changed {
				anyDirty = true
			}
		}
		if diffOutput && anyDirty {
			os.Exit(1)
		}
		return nil
	},
}

func loadConfig(path string) (*config.Config, error) {
	if path != "" {
		return config.Load(path)
	}
	// Try .columnar.json in current working directory.
	if _, err := os.Stat(".columnar.json"); err == nil {
		return config.Load(".columnar.json")
	}
	return config.Default(), nil
}

func formatFile(path string, cfg *config.Config) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	lang := languageFlag
	if lang == "" {
		lang = formatter.DetectLanguage(path)
	}
	out, err := formatter.Format(string(data), lang, cfg)
	if err != nil {
		return false, fmt.Errorf("format %s: %w", path, err)
	}
	changed := out != string(data)
	if diffOutput {
		if changed {
			fmt.Fprintf(os.Stderr, "would reformat: %s\n", filepath.Clean(path))
		}
		return changed, nil
	}
	if writeInPlace {
		if !changed {
			return false, nil
		}
		return changed, os.WriteFile(path, []byte(out), 0644)
	}
	_, err = io.WriteString(os.Stdout, out)
	return changed, err
}
