package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ger/redis-lite-go/internal/resp"
	"github.com/spf13/cobra"
)

//go:embed "commands-docs"
var commandsFS embed.FS

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorReset  = "\033[0m"
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
			if strings.HasPrefix(strings.ToLower(command), "help") {
				handleHelp(command)
				continue
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

func handleHelp(help string) {
	parts := strings.Fields(help)
	if len(parts) == 1 {
		fmt.Println("redis-lite-cli", version)
		fmt.Println("To get help about Redis command type:")
		fmt.Println("  \"help command\" for help on command")
		fmt.Println("  \"quit\" to exit")
		fmt.Print(host, ":", port, "> ")
		return
	}
	for _, p := range parts {
		if p == "help" {
			continue
		}
		printHelp(p)
	}
	fmt.Print(host, ":", port, "> ")
}

func printHelp(command string) {
	content, err := fs.ReadFile(commandsFS, filepath.Join("commands-docs", command+".json"))
	if err != nil {
		fmt.Println(err)
		fmt.Println("No known help for this command. Ask for online help")
		return
	}

	var data map[string]interface{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		fmt.Println(err)
		return
	}
	command = strings.ToUpper(command)
	if _, ok := data[command]; !ok {
		return
	}
	fmt.Println()
	if commandData, ok := data[command].(map[string]interface{}); ok {
		if summary, ok := commandData["summary"]; ok {
			fmt.Println(ColorYellow, "  summary: ", ColorReset, summary)
		}
		if group, ok := commandData["group"]; ok {
			fmt.Println(ColorYellow, "  group: ", ColorReset, group)

		}
		if since, ok := commandData["since"]; ok {
			fmt.Println(ColorYellow, "  since: ", ColorReset, since)
		}
	}
}

func printArguments(arguments []interface{}) {
	var ret string
	for _, arg := range arguments {
		if argMap, ok := arg.(map[string]interface{}); ok {
			fmt.Print("    - ")
			if name, ok := argMap["name"].(string); ok {
				fmt.Print("Name: ", name, ", ")
				if argType, ok := argMap["type"].(string); ok {
					if argType == "key" {
						ret += " " + name
					}
				}
				if argMultiple, ok := argMap["multiple"].(bool); ok {
					if argMultiple {
						ret += " [" + name + "...]"
					}
				}
			}
			if argType, ok := argMap["type"].(string); ok {
				fmt.Print("Type: ", argType)
				if argType == "block" {

				} else {
					ret += " " + argType
				}
			}
			if subArgs, ok := argMap["arguments"].([]interface{}); ok {
				fmt.Print(", Sub-Arguments: [")
				for i, subArg := range subArgs {
					if subArgMap, ok := subArg.(map[string]interface{}); ok {
						if i > 0 {
							fmt.Print("|")
						}
						if token, ok := subArgMap["token"].(string); ok {
							fmt.Print(token)
						}
					}
				}
				fmt.Print("]")
			}
			fmt.Println()
		}
	}
}
