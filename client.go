// client.go

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func startClient(nodeAddress []string) {

	reader := bufio.NewReader(os.Stdin)

	// 부트스트랩 노드로부터 받은 주소들에 연결
	if len(nodeAddress) > 0 {
		for _, address := range nodeAddress {

			if address == "" {
				fmt.Println("Empty node address, skipping...")
				continue
			}

			// 입력한 주소로 서버에 연결 시도
			conn, err := net.Dial("tcp", address)
			if err != nil {
				printError(fmt.Sprintf("Error reading from connection : %v", err))

			} else {
				fmt.Println("Successfully connected to", address)
				connectedPeers = append(connectedPeers, conn)
			}

		}
	} else {
		fmt.Println("No node addresses provided. Waiting for connection...")
	}

	// 각 피어와 병렬로 메시지 주고 받기 고루틴
	for _, conn := range connectedPeers {
		go handlePeerCommunication(conn) // 각 연결에 대해 별도의 고루틴으로 처리
	}

	// 모든 피어에 메시지 전송
	for {
		fmt.Print("Enter message: ")
		message, _ := reader.ReadString('\n')

		// 모든 피어에 메시지 전송
		for _, conn := range connectedPeers {
			if conn != nil {
				_, err := conn.Write([]byte(message))
				if err != nil {
					printError(fmt.Sprintf("Error sending message to peer: %v", err))
					connectedPeers = removeConn(connectedPeers, conn)
					conn.Close()
				}
			}
		}

	}
}

func handlePeerCommunication(conn net.Conn) {
	defer conn.Close()

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				printError(fmt.Sprintf("Peer disconnected: %s", conn.RemoteAddr().String()))
				connectedPeers = removeConn(connectedPeers, conn)

			} else {
				printError(fmt.Sprintf("Error reading from peer: %v", err))
			}
			return
		}

		message = strings.TrimSpace(message)
		printMessage(fmt.Sprintf("Message received from peer : %s from : %s", message, conn.RemoteAddr().String()))
	}
}
