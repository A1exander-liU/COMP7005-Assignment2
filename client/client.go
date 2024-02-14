package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

const (
	network string = "tcp"
	address string = "192.168.1.137:8081"
)

type Context struct {
	Address  string
	Port     string
	Ip       string
	Socket   net.Conn
	File     string
	Content  string
	Size     int
	Response string
	ExitCode int
}

func exit(context *Context) {
	os.Exit(context.ExitCode)
}

func connect(context *Context) {
	client, err := net.Dial("tcp", context.Address)
	if err != nil {
		fmt.Println("Socket Connect Error:\n", err)
		context.ExitCode = 1
		exit(context)
	}
	context.Socket = client
}

func readFile(context *Context) {
	content, err := os.ReadFile(context.File)
	if err != nil {
		fmt.Println("Read File Error:\n", err)
		HandleUserInput(context)
	}

	if len(content) == 0 {
		fmt.Println("File is empty")
		HandleUserInput(context)
	}

	context.Content = string(content)
	context.Size = len(context.Content)
}

func sendDataSize(context *Context) error {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, uint64(context.Size))

	_, err := context.Socket.Write(buffer)

	if err != nil {
		fmt.Println("Write Error:\n", err)
		return err
	}

	return nil
}

func sendData(context *Context) error {
	_, err := context.Socket.Write([]byte(context.Content))

	if err != nil {
		return err
	}

	return nil
}

func receiveDataSize(context *Context) error {
	buf := make([]byte, 8) // 8 bytes for an int64
	_, err := context.Socket.Read(buf)

	if err != nil {
		fmt.Println("Socket Read Error:\n", err)
		return err
	}

	context.Size = int(binary.BigEndian.Uint64(buf))
	return nil
}

func receiveData(context *Context) error {
	read := 0
	var dataBuffer bytes.Buffer
	buffer := make([]byte, 1024)

	for {
		nBytes, err := context.Socket.Read(buffer)
		read += nBytes

		if err != nil {
			fmt.Println("Socket Read Error:\n", err)
			return err
		}

		dataBuffer.Write(buffer[:nBytes])

		if read >= context.Size {
			break
		}
	}

	context.Response = dataBuffer.String()
	return nil
}

func receive(context *Context) {
	err := receiveDataSize(context)
	if err != nil {
		HandleUserInput(context)
	}

	err = receiveData(context)
	if err != nil {
		HandleUserInput(context)
	}
}

func send(context *Context) {
	err := sendDataSize(context)
	if err != nil {
		HandleUserInput(context)
	}

	err = sendData(context)
	if err != nil {
		HandleUserInput(context)
	}
}

func handleSendFile(context *Context) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter path to file: ")
	filePath, _ := reader.ReadString('\n')
	context.File = strings.TrimSpace(filePath)

	readFile(context)

	connect(context)
	defer context.Socket.Close()
	defer func(context *Context) { context.Socket = nil }(context)

	send(context)

	receive(context)

	fmt.Println(context.Response)
}

func HandleUserInput(context *Context) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("1. Send File\n2. Quit")
		choice, _ := reader.ReadString('\n')

		switch choice = strings.TrimSpace(choice); choice {
		case "1":
			handleSendFile(context)
		case "2":
			context.ExitCode = 0
			Exit(context)
		default:
			fmt.Println("Please enter 1 or 2")
		}
		fmt.Println()
	}
}

func sendFileMultiple(n int, file string) {
	var wg sync.WaitGroup
	wg.Add(n)

	done := 0

	for i := 0; i < n; i++ {
		go func(id int) {
			context := Context{}
			context.Ip = transformIp("2605:8d80:482:4c89:cbd3:4c92:514a:29b5")
			context.Port = "8081"
			context.Address = transformAddress(context.Ip, context.Port)
			context.File = file
			defer wg.Done()

			connect(&context)
			readFile(&context)

			sendDataSize(&context)
			sendData(&context)

			receiveDataSize(&context)
			receiveData(&context)

			fmt.Println(context.Response)
			done++
		}(i + 1)
	}

	wg.Wait()
}

func transformAddress(ip string, port string) string {
	return ip + ":" + port
}

func transformIp(ip string) string {
	index := strings.Index(ip, ":")
	if index >= 0 {
		return fmt.Sprintf("[%s]", ip)
	}
	return ip
}

func Exit(context *Context) {
	if context.Socket != nil {
		err := context.Socket.Close()

		if err != nil {
			println("Close Socket Error:\n", err)
			context.ExitCode = 1
			os.Exit(context.ExitCode)
		}
	}
	os.Exit(context.ExitCode)
}

func parseArgs(args []string, context *Context) {
	if len(args) < 3 {
		fmt.Println("Need to enter address and a port")
		context.ExitCode = 1
		Exit(context)
	} else {
		context.Ip = transformIp(args[1])
		context.Port = args[2]
		context.Address = transformAddress(context.Ip, context.Port)
		HandleUserInput(context)
	}
}

func main() {
	context := Context{}
	parseArgs(os.Args, &context)
	// sendFileMultiple(10, "hi.txt")
}
