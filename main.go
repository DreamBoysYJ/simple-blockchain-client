package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// 글로벌 변수 : 모든 피어의 연결 정보 저장하는 리스트
var connectedPeers []net.Conn

func main() {
	// 명령줄
	mode := flag.String("mode", "fullNode", "Start in 'Bootstrap Node' or 'FullNode' ")
	// 명령줄 인자 파싱 (flag.Parse() 필수)
	flag.Parse()

	serverListening := make(chan string)

	bootstrapAddress := "localhost:8282"

	if *mode == "bootstrap" {
		// startBootstrapServer(bootstrapAddress)
		bootstrapServer()

	} else if *mode == "fullNode" {
		// 1. 서버 실행
		go startServer(serverListening)

		// 2. 서버 주소 받아옴
		serverAddress := <-serverListening

		// 3. 부트스트랩 노드에 연결하고, 내 서버 정보 전달
		nodeAddress := connectBootstrapNode(bootstrapAddress, serverAddress)
		fmt.Println("부트스트랩 노드로 부터 받은 노드들 주소 :", nodeAddress)

		// 4. 받은 노드들과 연결 시도
		startClient(serverAddress, nodeAddress)
	} else {
		fmt.Println("Invalid mode. Use -mode=bootstrap or -mode=fullNode")
	}

}

func startServer(serverListening chan<- string) {

	// TCP 서버 시작
	min := 6666
	max := 8888
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

func startClient(serverAddress string, nodeAddress []string) {

	reader := bufio.NewReader(os.Stdin)
	// var err error

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
		printMessage(fmt.Sprintf("Message received from peer : %s", message))
	}
}
