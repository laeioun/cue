package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/laeioun/cue/internal/completion"
)

func specCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Manage completion specs",
	}

	var force bool
	generateCmd := &cobra.Command{
		Use:   "generate <command>",
		Short: "Generate a YAML spec from command help",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateSpec(cmd, args[0], force)
		},
	}
	generateCmd.Flags().BoolVar(&force, "force", false, "overwrite an existing generated spec")

	cmd.AddCommand(generateCmd)
	return cmd
}

func generateSpec(cmd *cobra.Command, cmdName string, force bool) error {
	root, err := completion.GenerateSpec(cmdName)
	if err != nil {
		return err
	}

	path, err := completion.UserSpecPath(cmdName)
	if err != nil {
		return err
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; rerun with --force to overwrite it", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	data, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
	return err
}
