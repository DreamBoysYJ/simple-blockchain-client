package main

import (
	"flag"
	"fmt"
	"net"
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
		fmt.Println("부트스트랩 노드로부터 받은 노드들 주소 :", nodeAddress)

		// 4. 받은 노드들과 연결 시도
		startClient(nodeAddress)
	} else {
		fmt.Println("Invalid mode. Use -mode=bootstrap or -mode=fullNode")
	}

}
