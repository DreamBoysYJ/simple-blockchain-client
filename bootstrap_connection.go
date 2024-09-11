// bootstrap_connection.go

package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// func connectBootstrapNode(bootstrapAddress string, serverAddress string) []string {

// 	var connectedNodes []string

// 	// 부트스트랩 노드에 연결
// 	conn, err := net.Dial("tcp", bootstrapAddress)
// 	if err != nil {
// 		printError(fmt.Sprintf("Error connecting to bootstrap node: %v", err))
// 		return connectedNodes
// 	}
// 	defer conn.Close()

// 	// 내 서버 주소를 부트스트랩 노드에 전달
// 	fmt.Println("Sending server address to bootstrap node : ", serverAddress)

// 	const Neighbors = 0x01
// 	protocol := []byte{Neighbors}
// 	myAddress := []byte(serverAddress)
// 	protocol = append(protocol, myAddress...)

// 	_, err = conn.Write(protocol)
// 	// _, err = conn.Write([]byte(serverAddress + "\n"))
// 	if err != nil {
// 		printError(fmt.Sprintf("Error sending server address: %v", err))
// 	}

// 	// 부트스트랩 노드로부터 다른 노드들 주소 받기
// 	message, err := bufio.NewReader(conn).ReadString('\n')
// 	if err != nil {
// 		if err == io.EOF {
// 			printError("Bootstrap node has closed the connection.")

// 		} else {
// 			printError(fmt.Sprintf("Error reading from bootstrap node: %v", err))
// 		}
// 		return connectedNodes
// 	}

// 	// 받은 메시지를 자료 구조에 저장
// 	connectedNodes = strings.Split(strings.TrimSpace(message), ",")
// 	fmt.Println("부트스트랩 노드로 부터 받은 노드들: ", connectedNodes)
// 	return connectedNodes

// }

func connectBootstrapNode(bootstrapAddress string, serverAddress string) []string {
	var nodeLists []string

	// create UDP address, socket
	bootstrapUDPAddr, err := net.ResolveUDPAddr("udp", bootstrapAddress)
	if err != nil {
		printError(fmt.Sprintf("Error resolving UDP addres : %v", err))
		return nodeLists
	}

	conn, err := net.DialUDP("udp", nil, bootstrapUDPAddr)
	if err != nil {
		printError(fmt.Sprintf("Error connecting to bootstrap node : %v", err))
		return nodeLists
	}
	defer conn.Close()

	// 1. Send Ping
	fmt.Println("Sending Ping to bootstrap node")
	pingMessage := []byte{Ping}
	_, err = conn.Write(pingMessage)
	if err != nil {
		printError(fmt.Sprintf("Error sending Ping message : %v", err))
		return nodeLists
	}

	// 2. Waiting Pong
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		printError(fmt.Sprintf("Error receiving Pong message : %v", err))
		return nodeLists
	}

	// 3. Send FindNode
	if buffer[0] == Pong {
		fmt.Println("Received Pong message")
		fmt.Println("Sending FindNode message to bootstrap node")
		findNodeMessage := append([]byte{FindNode}, []byte(serverAddress)...)
		_, err = conn.Write(findNodeMessage)
		if err != nil {
			printError(fmt.Sprintf("Error sending FindNode message : %v", err))
			return nodeLists
		}

		// 4. Waiting Neighbors
		n, _, err = conn.ReadFromUDP(buffer)
		if err != nil {
			printError(fmt.Sprintf("Error receiving Neighbors message : %v", err))
			return nodeLists
		}

		if buffer[0] == Neighbors {
			message := string(buffer[1:n])
			nodeLists = strings.Split(strings.TrimSpace(message), ",")
			// fmt.Println("부트스트랩 노드로부터 받은 노드들 : ", nodeLists)
			return nodeLists

		} else {
			printError("Did not receive Neighbors message")
			return nodeLists
		}

	} else {
		printError("Did not receive Pong message")
		return nodeLists
	}

}
