// cmd/quick_assist.go

package cmd

import (
	"fmt"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

var (
	dceEnabled bool
)

var quickAssistCmd = &cobra.Command{
	Use:   "quickassist [query]",
	Short: "Get quick assistance from the LLM",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		// Check if DCE is enabled
		dceEnabled, _ := cmd.Flags().GetBool("dce")

		if dceEnabled {
			// Handle QuickAssist with DCE
			response, err := llm.HandleExtensionQuickAssist("", query, true)
			if err != nil {
				fmt.Printf("Error with DCE-enabled QuickAssist: %v\n", err)
				return
			}
			fmt.Println("\nQuickAssist (DCE Enabled) Response:")
			fmt.Println(response)
		} else {
			// Handle standard QuickAssist
			response, err := llm.HandleCLIQuickAssist(query)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

			fmt.Println("\nQuickAssist Response:")
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(quickAssistCmd)
	quickAssistCmd.Flags().BoolP("serve", "u", false, "Start HTTP server for extension integration")
	quickAssistCmd.Flags().BoolP("dce", "d", false, "Enable Dynamic Context Engine")
}
