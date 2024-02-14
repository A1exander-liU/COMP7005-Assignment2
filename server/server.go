package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

var connections = 0

func formatResponse(wordCount int, charCount int, freqs map[rune]int) string {
	message := fmt.Sprintf("Word Count: %d | Char Count: %d\n", wordCount, charCount)

	message += "Character Frequencies\n"

	for key, value := range freqs {
		message += fmt.Sprintf("%c: %d/%d\n", key, value, charCount)
	}

	return message
}

func characterFrequencies(text string) map[rune]int {
	regex := regexp.MustCompile(`\s+`)
	cleaned := regex.ReplaceAllString(text, "")
	cleaned = strings.ToLower(cleaned)

	freqs := make(map[rune]int)
	for _, value := range cleaned {
		freqs[value]++
	}
	return freqs
}

func characterCount(text string) int {
	regex := regexp.MustCompile(`\s+`)
	cleaned := regex.ReplaceAllString(text, "")
	return len(strings.TrimSpace(cleaned))
}

func wordCount(text string) int {
	regex := regexp.MustCompile(`\s+`)
	cleaned := regex.ReplaceAllString(text, " ")
	cleaned = strings.TrimSpace(cleaned)

	if len(cleaned) == 0 {
		return 0
	}

	return len(strings.Split(cleaned, " "))
}

func sendDataSize(conn net.Conn, size int) error {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, uint64(size))
	_, err := conn.Write(buffer)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func sendData(conn net.Conn, data string) error {
	fmt.Println("Sending Back\n", data)
	_, err := conn.Write([]byte(data))
	if err != nil {
		return err
	}
	return nil
}

func receiveDataSize(conn net.Conn) int {
	buf := make([]byte, 8) // 8 bytes for an int64
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	return int(binary.BigEndian.Uint64(buf))
}

func receiveData(conn net.Conn, size int) string {
	read := 0
	var dataBuffer bytes.Buffer
	buffer := make([]byte, 1024)

	for {
		nBytes, err := conn.Read(buffer)
		read += nBytes

		if err != nil {
			fmt.Println("Error:", err)
			return ""
		}

		dataBuffer.Write(buffer[:nBytes])

		if read >= size {
			break
		}
	}

	return dataBuffer.String()
}

func listen(address string) net.Listener {
	server, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Failed to listen:\n", err)
		os.Exit(1)
	}
	fmt.Printf("Server listening on %s\n", address)
	return server
}

func send(conn net.Conn, content string) {
	err := sendDataSize(conn, len(content))
	if err != nil {
		fmt.Println("Send Error:\n", err)
		return
	}

	err = sendData(conn, content)
	if err != nil {
		fmt.Println("Send Error:\n", err)
	}
}

func handle(connection net.Conn) {
	size := receiveDataSize(connection)
	content := receiveData(connection, size)

	wCount := wordCount(content)
	cCount := characterCount(content)
	freqs := characterFrequencies(content)

	response := formatResponse(wCount, cCount, freqs)

	send(connection, response)
}

func acceptConnections(server net.Listener, channel chan net.Conn, quit chan os.Signal) {
	for {
		conn, err := server.Accept()
		if err != nil {
			continue
		}
		connections++
		fmt.Printf("Client %d connected\n", connections)

		channel <- conn
	}
}

func cleanup(conn net.Listener) {
	if conn != nil {
		err := conn.Close()
		if err != nil {
			fmt.Println("Close Error:\n", err)
			os.Exit(1)
		}

		println("Server closed successfully")
	}
	println("Exiting")
	os.Exit(0)
}

func handleSigInt(channel chan os.Signal, exit func(net.Listener), conn net.Listener) {
	for {
		sig := <-channel

		switch sig {
		case os.Interrupt, syscall.SIGINT:
			exit(conn)
		}
	}
}

func transformAddress(address string) string {
	ip6Index := strings.Index(address, ":")
	if ip6Index > -1 {
		return "[" + address + "]"
	}
	return address
}

func main() {
	server := listen("[::]:8081")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT)

	go handleSigInt(sigChan, cleanup, server)

	incoming := make(chan net.Conn)

	go acceptConnections(server, incoming, sigChan)

	for {
		select {
		case conn := <-incoming:
			go handle(conn)
		}
	}
}
