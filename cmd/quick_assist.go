// cmd/quickassist.go

package cmd

import (
	"fmt"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/llm"
	"github.com/spf13/cobra"
)

var quickAssistCmd = &cobra.Command{
	Use:   "quickassist [query]",
	Short: "Get quick assistance from the LLM",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := strings.Join(args, " ")

		// Handle direct terminal request
		response, err := llm.HandleCLIQuickAssist(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Println("\nQuickAssist Response:")
		fmt.Println(response)
	},
}

func init() {
	rootCmd.AddCommand(quickAssistCmd)
	quickAssistCmd.Flags().BoolP("serve", "u", false, "Start HTTP server for extension integration")
}
