package cli

import (
	"fmt"
	"io/fs"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/laeioun/cue/internal/aliases"
	"github.com/laeioun/cue/internal/completion"
	"github.com/laeioun/cue/internal/picker"
)

func Execute(specs fs.FS) error {
	return New(specs).Execute()
}

func New(specs fs.FS) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cue",
		Short: "Tab completion picker for command-line tools",
	}

	completeCmd := &cobra.Command{
		Use:   "complete <buffer> <cursor>",
		Short: "Return completions for a shell buffer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			line := args[0]
			cursor, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}

			workingLine, workingCursor := line, cursor
			aliasMap, err := aliases.Load()
			if err == nil {
				workingLine, workingCursor, _ = aliases.Expand(line, cursor, aliasMap)
			}

			completions, err := completion.Complete(specs, workingLine, workingCursor)
			if err != nil || len(completions) == 0 {
				if workingLine != line {
					fmt.Println(workingLine)
				}
				return nil
			}

			_, partial := completion.Parse(workingLine, workingCursor)
			if selected, ok := completion.FastSelection(completions, partial); ok {
				fmt.Println(completion.ApplyCompletion(workingLine, workingCursor, selected))
				return nil
			}

			selected, err := picker.Run(completions)
			if err != nil {
				return nil
			}
			if selected != "" {
				fmt.Println(completion.ApplyCompletion(workingLine, workingCursor, selected))
			}
			return nil
		},
	}

	rootCmd.AddCommand(completeCmd)
	return rootCmd
}
