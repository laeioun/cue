package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laeioun/cue/internal/config"
)

func vimCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vim [on|off|toggle|status]",
		Short: "Configure vim-style picker navigation",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "status"
			if len(args) > 0 {
				action = args[0]
			}
			return configureVimMode(cmd, action)
		},
	}
}

func configureVimMode(cmd *cobra.Command, action string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch action {
	case "on", "enable", "enabled":
		cfg.VimMode = true
		if err := config.Save(cfg); err != nil {
			return err
		}
	case "off", "disable", "disabled":
		cfg.VimMode = false
		if err := config.Save(cfg); err != nil {
			return err
		}
	case "toggle":
		cfg.VimMode = !cfg.VimMode
		if err := config.Save(cfg); err != nil {
			return err
		}
	case "status":
	default:
		return fmt.Errorf("unknown vim mode action %q (expected on, off, toggle, or status)", action)
	}

	state := "off"
	if cfg.VimMode {
		state = "on"
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "vim mode %s\n", state)
	return err
}
