// server.go

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

func startServer(serverListening chan<- string) {

	// TCP 서버 시작
	min := 6666
	max := 9999
	randomPort := rand.Intn(max-min+1) + min
	address := "localhost:" + strconv.Itoa(randomPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting server:", err)
		printError(fmt.Sprintf("Error starting server: %v", err))
		return
	}

	fmt.Println("Server is listening on port", randomPort)
	serverListening <- address

	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			printError(fmt.Sprintf("Error accepting connection: %v", err))
			continue
		}
		printMessage(fmt.Sprintf("Peer connected from: %v", conn.RemoteAddr().String()))

		// 연결된 피어를 글로벌 변수에 저장
		connectedPeers = append(connectedPeers, conn)

		// 피어와 통신 처리
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			printError(fmt.Sprintf("Error reading from connection : %v", err))
			return
		}
		message = strings.TrimSpace(message)
		printMessage(fmt.Sprintf("Message received : %s", message))
	}
}
