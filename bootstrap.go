// bootstrap.go

package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// 메시지 타입 상수 정의
const (
	Ping      = 0x01
	Pong      = 0x02
	FindNode  = 0x03
	Neighbors = 0x04
)

// startBootstrapServer : 부트스트랩 노드를 실행하는 함수
func startBootstrapServer(address string) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting bootstrap node:", err)
		return
	}
	fmt.Println("Bootstrap Node is listening on port:", address)
	defer listener.Close()

	// 연결된 노드들의 주소를 저장할 슬라이스
	var connectedNodes []string

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		// buf := make([]byte, 1024)
		// n, err := conn.Read(buf)
		// if err != nil {
		// 	fmt.Println("Error reading from connection: ", err)
		// 	return
		// }

		// if n == 0 {
		// 	fmt.Println("No data received")
		// 	return
		// }

		// messageType := buf[0]

		// switch messageType {
		// case Ping:
		// 	fmt.Println("Received Ping message")
		// 	additionalData := strings.TrimSpace(string(buf[1:n]))
		// 	fmt.Println("Additional data : ", additionalData)
		// }

		// 노드로부터 서버 주소 수신
		nodeAddress, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from node:", err)
			conn.Close()
			continue
		}

		nodeAddress = strings.TrimSpace(nodeAddress)
		fmt.Println("새로운 노드가 연결되었습니다, 주소:", nodeAddress)

		// // 기존 노드들 정보 전달
		connectedNodesString := strings.Join(connectedNodes, ",")
		if len(connectedNodes) > 0 {
			_, err := conn.Write([]byte(connectedNodesString + "\n"))
			if err != nil {
				fmt.Println("Error sending nodes address:", err)
			}
		}

		// 새로운 노드를 목록에 추가
		connectedNodes = append(connectedNodes, nodeAddress)

		// 데이터 전송 후 연결 닫음
		conn.Close()
		fmt.Println("데이터를 성공적으로 전달 후 연결이 끊겼습니다. 주소:", nodeAddress)

	}
}

func bootstrapServer() {

	// 연결된 노드들의 주소를 저장할 슬라이스
	var connectedNodes []string

	// create UDP address (localhost:8282)
	addr := net.UDPAddr{
		Port: 8282,
		IP:   net.ParseIP("127.0.0.1"),
	}

	// Waiting for UDP connection
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error starting UDP server :", err)
		return
	}

	fmt.Println("Bootstrap Node is listening on port:", fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port))

	defer conn.Close()

	// create Buffer
	buf := make([]byte, 1024)

	// Waiting for data
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving UDP data : ", err)
			continue
		}
		messageType := buf[0]

		switch messageType {
		case Ping:
			fmt.Printf("Received Ping message from %s\n", remoteAddr.String())
			_, err = conn.WriteToUDP([]byte{Pong}, remoteAddr)
			if err != nil {
				fmt.Println("Error sending UDP response:", err)
			}
			fmt.Printf("Sent Pong message to %s\n", remoteAddr.String())
		case FindNode:
			fmt.Printf("Received FindNode message from %s\n", remoteAddr.String())

			nodeInfo := string(buf[1:n])

			connectedNodesString := strings.Join(connectedNodes, ",")
			message := append([]byte{Neighbors}, []byte(connectedNodesString)...)
			_, err = conn.WriteToUDP(message, remoteAddr)
			if err != nil {
				fmt.Println("Error sending Neighbors protocol : ", err)
			}
			fmt.Printf("Sent Neighbor message to %s\n", remoteAddr.String())

			// 새로운 노드를 목록에 추가
			connectedNodes = append(connectedNodes, nodeInfo)
			// fmt.Printf("Saved nodeInfo complete", nodeInfo)
			fmt.Println("Saved nodeInfo complete")

		}
	}
}
