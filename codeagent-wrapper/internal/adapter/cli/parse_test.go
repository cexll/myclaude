package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestBuildSingleConfigResumeMode(t *testing.T) {
	opts := &Options{}
	cmd := &cobra.Command{SilenceErrors: true, SilenceUsage: true, Args: cobra.ArbitraryArgs}
	AddRootFlags(cmd.Flags(), opts)
	args := []string{"resume", "sess-1", "do work", "/tmp/work"}

	cfg, err := BuildSingleConfig(cmd, args, args, opts, viper.New())
	if err != nil {
		t.Fatalf("BuildSingleConfig() error = %v", err)
	}
	if cfg.Mode != "resume" || cfg.SessionID != "sess-1" || cfg.Task != "do work" || cfg.WorkDir != "/tmp/work" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
