package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ger/redis-lite-go/internal/resp"
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
	input := make(chan string)

	writer := resp.NewRespWriter(conn)
	respReader := resp.NewRespReader(conn)

	// listen to stop signal
	go func() {
		<-sigs
		done <- true
	}()

	// go routine to listen for input
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print(host, ":", port, "> ")
		for {
			if scanner.Scan() {
				line := scanner.Text()
				if line == "\x0c" { //Ctrl-L
					clearScreen()
				} else {
					input <- line
				}
			} else {
				done <- true
				break
			}
		}
	}()

	for {
		select {
		case <-done:
			fmt.Println()
			return
		case command := <-input:
			if strings.ToLower(command) == "exit" || strings.ToLower(command) == "quit" {
				return
			}

			parts := strings.Fields(command)

			var cmdPayload resp.Payload
			for _, part := range parts {
				p := resp.Payload{DataType: string(resp.BULKSTRING), Bulk: part}
				cmdPayload.Array = append(cmdPayload.Array, p)
			}

			cmdPayload.DataType = string(resp.ARRAY)

			// Send the command to Redis
			err := writer.Write(&cmdPayload)
			if err != nil {
				fmt.Println("(error) Err writing array:", err)

			}

			// Read the response
			cmd, err := respReader.Read()
			if err != nil {
				fmt.Println("(error) Err reading response:", err)
			} else if cmd.DataType == "" {
				fmt.Println("(nil)")
			}

			// Print the response
			printRedisServerAnswer(cmd)
			fmt.Print(host, ":", port, "> ")

			// TODO format request and response
		}
	}
}

func printRedisServerAnswer(cmd resp.Payload) {
	switch cmd.DataType {
	case string(resp.ERROR):
		fmt.Println("Error:", cmd.Str)
	case string(resp.BULKSTRING):
		fmt.Println(cmd.Bulk)
	case string(resp.INTEGER):
		fmt.Println("(integer) ", cmd.Num)
	case string(resp.STRING):
		fmt.Println(cmd.Str)
	default:
		fmt.Println("unsupported answer format")
	}
}

func clearScreen() {
	// Only working on linux
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
	fmt.Print(host, ":", port, "> ")
}
