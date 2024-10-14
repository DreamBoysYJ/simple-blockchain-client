package main

import (
	"flag"
	"fmt"

	"simple_p2p_client/bootnode"
	"simple_p2p_client/p2p"
	rpcserver "simple_p2p_client/rpc-server"
	"simple_p2p_client/utils"
)

func main() {
	// 명령줄
	mode := flag.String("mode", "fullnode", "Start in 'Bootstrap Node' or 'FullNode' ")
	port := flag.Int("port", 30303, "The port on which the server listen (TCP & UDP)")
	rpcPort := flag.Int("rpcport", 8080, "The port on which the RPC server listens")
	// 명령줄 인자 파싱 (flag.Parse() 필수)
	flag.Parse()

	tcpAddress := make(chan string)
	udpAddress := make(chan string)
	bootstrapAddress := "localhost:8282"

	if *mode == "bootnode" {
		bootnode.StartBootstrapServer()

	} else if *mode == "fullnode" {

		// DB 오픈

		// RPC 서버 시작
		go rpcserver.StartRpcServer(*rpcPort)

		// 1. TCP 서버 실행

		go p2p.StartTCPServer(tcpAddress, *port)

		// 2. UDP 서버 실행
		go p2p.StartUDPServer(udpAddress, tcpAddress, *port)

		// 3. UDP 서버 주소 받아옴
		udpServerAddress := <-udpAddress

		// 4. 부트스트랩 노드에 연결하고, 내 UDP 주소 전달
		nodeAddress, err := p2p.ConnectBootstrapNode(bootstrapAddress, udpServerAddress)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Failed to connect to bootstrap node: %v", err))
			return
		}
		fmt.Println("부트스트랩 노드로부터 받은 노드들 주소 :", nodeAddress)

		// 5. 부트스트랩 노드로 부터 받은 노드들과 피어 연결 시도
		p2p.StartClient(nodeAddress)
	} else {
		fmt.Println("Invalid mode. Use -mode=bootstrap or -mode=fullNode")
	}

}
