package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

var (
	buildTime string
	version   string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the CLI",
	Long:  `All software has versions. This is CLI's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("CLI version:", version)
		fmt.Println("CLI buildTime:", buildTime)
	},
}

func main() {

	rootCmd := &cobra.Command{
		Use:   "redis-lite-cli",
		Short: "Redis CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello from redis-cli")
		},
	}

	rootCmd.AddCommand(versionCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
