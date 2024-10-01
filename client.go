// client.go

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

func startClient(nodeAddress []string) {

	reader := bufio.NewReader(os.Stdin)

	// NodeDiscovery (Peer와 연결 직전까지)
	if len(nodeAddress) > 0 {
		for _, address := range nodeAddress {

			if address == "" {
				fmt.Println("Empty node address, skipping...")
				continue
			}

			nodeUDPAddr, err := net.ResolveUDPAddr("udp", address)
			if err != nil {
				printError(fmt.Sprintf("Error resolving UDP address : %v", err))
				return
			}
			conn, err := net.DialUDP("udp", nil, nodeUDPAddr)
			if err != nil {
				printError(fmt.Sprintf("Error connecting to node : %v", err))
				return
			}

			// 1. Send Ping
			fmt.Println("Sending Ping to node")
			pingMessage := []byte{NodeDiscoveryPing}
			_, err = conn.Write(pingMessage)
			if err != nil {
				printError(fmt.Sprintf("Error sending Ping message : %v", err))
				return
			}

			// 2. Waiting Pong
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				printError(fmt.Sprintf("Error receiving Pong message : %v", err))
				return
			}
			// 3. Send ENRRequest
			if buffer[0] == NodeDiscoveryPong {
				fmt.Println("Received Pong message")
				fmt.Println("Sending ENRRequest")
				_, err = conn.Write([]byte{NodeDiscoveryENRRequest})
				if err != nil {
					printError(fmt.Sprintf("Error sending ENRRequest : %v", err))
					return
				}

				// 4. Waiting ENRResponse
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				n, _, err = conn.ReadFromUDP(buffer)
				if err != nil {
					printError(fmt.Sprintf("Error receiving ENRResponse : %v", err))
					return
				}

				if buffer[0] == NodeDiscoveryENRResponse {
					tcpServer := string(buffer[1:n])
					fmt.Println("TCP SERVER :::", tcpServer)
					// 3. TCP 연결

					// 주소로 UDP Ping 보내기
					conn, err := net.Dial("tcp", tcpServer)
					if err != nil {
						printError(fmt.Sprintf("Error reading from connection : %v", err))

					} else {
						fmt.Println("Successfully connected to", tcpServer)
						connectedPeers = append(connectedPeers, conn)
					}

				}

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
