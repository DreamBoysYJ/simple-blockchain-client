// server.go

package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// 메시지 타입 상수 정의
const (
	// Node Discovery 상수
	NodeDiscoveryENRRequest  = 0x05
	NodeDiscoveryENRResponse = 0x06

	// RLPx 상수
	RLPx = 0x07
)

// TCP 서버 실행
func startTCPServer(serverListening chan<- string, port int) {

	// TCP 서버 시작
	address := "localhost:" + strconv.Itoa(port)
	// min := 6666
	// max := 9999
	// randomPort := rand.Intn(max-min+1) + min

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error starting server:", err)
		printError(fmt.Sprintf("Error starting server: %v", err))
		return
	}

	fmt.Println("Server is listening on port", port)
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

// UDP 서버 실행
func startUDPServer(port int) {

	// 1. Create UDP address
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP("127.0.0.1"),
	}

	// 2. Waiting for UDP connection
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Error starting UDP server :", err)
		return
	}

	fmt.Println("UDP Server is listening on ", fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port))

	defer conn.Close()

	// create Buffer
	buf := make([]byte, 1024)

	// Listening UDP Server
	// Node Discovery Protocol
	for {
		// n : 바이트 수 (일단 생략)
		_, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receving UDP data : ", err)
			continue
		}

		messageType := buf[0]

		switch messageType {
		// Node가 payload에 Ping(0x01)을 보냈을 시
		case NodeDiscoveryPing:
			fmt.Printf("Received Ping message from %s\n", remoteAddr.String())
			// Pong으로 응답
			_, err = conn.WriteToUDP([]byte{NodeDiscoveryPong}, remoteAddr)
			if err != nil {
				fmt.Println("Error sending UDP response : ", err)
			}
			fmt.Printf("Sent Pong to %s\n", remoteAddr.String())
		// Node가 payload에 ENRRequest(0x05)를 보냈을 시
		case NodeDiscoveryENRRequest:
			fmt.Printf("Received ENRRequest from %s\n", remoteAddr.String())
			// ENRResponse(0x06)로 응답
			message := append([]byte{NodeDiscoveryENRResponse}, []byte("enode")...)
			_, err := conn.WriteToUDP(message, remoteAddr)
			if err != nil {
				fmt.Println("Error sending UDP message : ", err)
			}
			fmt.Printf("Send ENRResponse to %s\n", remoteAddr.String())

		}

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
		printMessage(fmt.Sprintf("Message received : %s, from :: %v\n", message, conn.RemoteAddr().String()))
	}
}
