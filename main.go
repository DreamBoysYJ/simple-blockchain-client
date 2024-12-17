package main

import (
	"flag"
	"fmt"
	"os"

	"simple_p2p_client/blockchain"
	"simple_p2p_client/bootnode"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/p2p"
	rpcserver "simple_p2p_client/rpc-server"
	"simple_p2p_client/utils"
)

func main() {
	// 명령줄
	mode := flag.String("mode", "fullnode", "Start in 'Bootstrap Node' or 'FullNode' ")
	port := flag.Int("port", 30303, "The port on which the server listen (TCP & UDP)")
	rpcPort := flag.Int("rpcport", 8080, "The port on which the RPC server listens")
	nodeID := flag.String("nodeID", "default", "Unique node identifier")
	// 명령줄 인자 파싱 (flag.Parse() 필수)
	flag.Parse()

	tcpAddress := make(chan string)
	udpAddress := make(chan string)
	bootstrapAddress := "localhost:8282"

	// nodeID 별 경로 설정
	dbPath := fmt.Sprintf("./db/%s", *nodeID)
	if err := ensureDBDirectory(dbPath); err != nil {
		fmt.Printf("Failed to prepare DB directory : %v/n", err)
		os.Exit(1)
	}

	// Open DB
	leveldb.SetDBPath(dbPath)
	_, err := leveldb.GetDBInstance()
	if err != nil {
		fmt.Printf("Failed to open DB for node %s: %v\n", *nodeID, err)
		os.Exit(1)
	}
	// DB 정리
	defer func() {
		if leveldb.IsDBOpened() {
			if err := leveldb.CloseDB(); err != nil {
				fmt.Printf("Failed to close DB: %v\n", err)
			}
		}
	}()

	if *mode == "bootnode" {
		bootnode.StartBootstrapServer()

	} else if *mode == "fullnode" {

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

		go blockchain.StartBlockchainProcessor()

		// 5. 부트스트랩 노드로 부터 받은 노드들과 피어 연결 시도
		p2p.StartClient(nodeAddress)
	} else {
		fmt.Println("Invalid mode. Use -mode=bootstrap or -mode=fullNode")
	}

}

// DB 디렉토리를 확인하고 없으면 생성
func ensureDBDirectory(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		err := os.MkdirAll(dbPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create DB directory : %v", err)
		}
	}
	return nil
}
