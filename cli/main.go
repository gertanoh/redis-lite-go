package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	buildTime string
	version   string
	host      string
	port      string
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

var rootCmd *cobra.Command

func main() {

	rootCmd = &cobra.Command{
		Use:   "redis-lite-cli",
		Short: "Redis CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello from redis-cli")
			address := fmt.Sprintf("%s:%s", host, port)

			conn, err := net.Dial("tcp", address)
			if err != nil {
				log.Fatal(err)
			}

			defer conn.Close()

			WaitForInput(host, port, conn)
		},
	}

	rootCmd.PersistentFlags().StringVar(&host, "host", "127.0.0.1", "Host to connect to")
	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "6379", "port to connect on")
	rootCmd.AddCommand(versionCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func WaitForInput(host, port string, conn net.Conn) {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT)
	done := make(chan bool, 1)

	// listen to stop signal
	go func() {
		<-sigs
		fmt.Println("receivied sig")
		done <- true
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-done:
			return
		default:
			fmt.Print(host, ":", port, ">")

			scanner.Scan()
			// TODO use go routine and channels to get here when there is command
			command := scanner.Text()

			if strings.ToLower(command) == "exit" || strings.ToLower(command) == "quit" {
				return
			}

			fmt.Println("command : ", command)
			// Send the command to Redis
			fmt.Fprintf(conn, command+"\r\n")

			// Read the response
			response, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Error reading response:", err)
				continue
			}

			// Print the response
			fmt.Println(response)

			// TODO format request and response
		}
	}

}
