package cli

import (
	"fmt"
	"io/fs"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/laeioun/cue/internal/aliases"
	"github.com/laeioun/cue/internal/completion"
	"github.com/laeioun/cue/internal/config"
	"github.com/laeioun/cue/internal/picker"
)

const cursorResponsePrefix = "__cue_cursor__:"

func Execute(specs fs.FS) error {
	return New(specs).Execute()
}

func New(specs fs.FS) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cue",
		Short: "Tab completion picker for command-line tools",
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

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

			cfg, _ := config.Load()
			result, err := picker.Run(completions, picker.Options{
				Query:   partial,
				VimMode: cfg.VimMode,
			})
			if err != nil {
				return nil
			}
			switch result.Action {
			case picker.ActionSelect:
				fmt.Println(completion.ApplyCompletion(workingLine, workingCursor, result.Selected))
			case picker.ActionBackspace:
				nextLine, nextCursor := completion.DeleteBeforeCursor(workingLine, workingCursor)
				fmt.Println(formatCursorResponse(nextLine, nextCursor))
			case picker.ActionDelete:
				nextLine, nextCursor := completion.DeleteAtCursor(workingLine, workingCursor)
				fmt.Println(formatCursorResponse(nextLine, nextCursor))
			case picker.ActionInsert:
				nextLine, nextCursor := completion.InsertAtCursor(workingLine, workingCursor, result.Text)
				fmt.Println(formatCursorResponse(nextLine, nextCursor))
			case picker.ActionDeleteWordBefore:
				nextLine, nextCursor := completion.DeleteWordBeforeCursor(workingLine, workingCursor)
				fmt.Println(formatCursorResponse(nextLine, nextCursor))
			case picker.ActionDeleteWordAt:
				nextLine, nextCursor := completion.DeleteWordAtCursor(workingLine, workingCursor)
				fmt.Println(formatCursorResponse(nextLine, nextCursor))
			case picker.ActionMoveLeft:
				fmt.Println(formatCursorResponse(workingLine, completion.MoveCursorLeft(workingLine, workingCursor)))
			case picker.ActionMoveRight:
				fmt.Println(formatCursorResponse(workingLine, completion.MoveCursorRight(workingLine, workingCursor)))
			case picker.ActionMoveWordLeft:
				fmt.Println(formatCursorResponse(workingLine, completion.MoveCursorWordLeft(workingLine, workingCursor)))
			case picker.ActionMoveWordRight:
				fmt.Println(formatCursorResponse(workingLine, completion.MoveCursorWordRight(workingLine, workingCursor)))
			}
			return nil
		},
	}

	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(specCmd())
	rootCmd.AddCommand(vimCmd())
	return rootCmd
}

func formatCursorResponse(line string, cursor int) string {
	return fmt.Sprintf("%s%d:%s", cursorResponsePrefix, cursor, line)
}
