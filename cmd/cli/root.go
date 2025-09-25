package main

import (
	"log"
	"os"

	"github.com/clems4ever/big-context/internal/cli"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mapred-llm <prompt> <data-file-path>",
	Short: "Command that performs a sort of map reduce on data in a file and using ChatGPT as the filter and reducer",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		prompt, dataFilePath := args[0], args[1]
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Panic("OPENAI_API_KEY environment variable must be set")
		}

		err := cli.Process(cmd.Context(), apiKey, cli.ModelGPT5Nano, prompt, dataFilePath)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func main() {
	rootCmd.Execute()
}
