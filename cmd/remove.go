package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zhenchaochen/kronos/internal/config"
)

var removeYes bool

func init() {
	removeCmd.Flags().BoolVarP(&removeYes, "yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a job from the config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := cfgPath
		if path == "" {
			path = config.DefaultConfigPath()
		}

		loaded, err := config.Load(path)
		if err != nil {
			return err
		}

		idx := -1
		for i, j := range loaded.Jobs {
			if j.Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("job %q not found", name)
		}

		if !removeYes {
			fmt.Printf("Remove job %q? [y/N] ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
				fmt.Println("Aborted.")
				return nil
			}
		}

		loaded.Jobs = append(loaded.Jobs[:idx], loaded.Jobs[idx+1:]...)

		if err := config.Save(path, loaded, nil); err != nil {
			return err
		}

		fmt.Printf("Removed job %q\n", name)
		return nil
	},
}
