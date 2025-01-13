package main

import (
	"flag"
	"fmt"
	"os"

	"simple_p2p_client/blockchain"
	"simple_p2p_client/bootnode"
	"simple_p2p_client/constants"
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
	bootstrapAddress := constants.BootstrapNodeAddress

	// nodeID 별 경로 설정
	// nodeID 별 절대 경로 설정
	dbPath := fmt.Sprintf("./db/%s", *nodeID)

	leveldb.SetDBPath(dbPath)

	// DB 초기화
	if err := leveldb.InitDB(); err != nil {
		fmt.Printf("Failed to initialize DB for node %s: %v\n", *nodeID, err)
		os.Exit(1)
	}
	defer func() {
		if err := leveldb.CleanupDB(); err != nil {
			fmt.Printf("Failed to cleanup DB: %v\n", err)
		}
	}()

	if *mode == "bootnode" {
		bootnode.StartBootstrapServer()

	} else if *mode == "fullnode" {
		// FullNode 초기화 로직
		initializeFullNode(*port, *rpcPort, bootstrapAddress, tcpAddress, udpAddress)
	} else {
		fmt.Println("Invalid mode. Use -mode=bootstrap or -mode=fullNode")
	}

}

func initializeFullNode(port, rpcPort int, bootstrapAddress string, tcpAddress, udpAddress chan string) {
	// 노드 계정 초기화
	if err := blockchain.InitializeNodeAccount(); err != nil {
		fmt.Printf("Failed to initialize node account: %v\n", err)
		os.Exit(1)
	}

	// Blockchain 초기화
	if err := blockchain.InitGenesisBlock(); err != nil {
		fmt.Printf("Failed to initialize blockchain: %v\n", err)
		os.Exit(1)
	}

	blockchain.InitMempool()
	go rpcserver.StartRpcServer(rpcPort)
	go p2p.StartTCPServer(tcpAddress, port)
	go p2p.StartUDPServer(udpAddress, tcpAddress, port)

	udpServerAddress := <-udpAddress
	nodeAddress, err := p2p.ConnectBootstrapNode(bootstrapAddress, udpServerAddress)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Failed to connect to bootstrap node: %v", err))
		return
	}

	fmt.Println("[Node Discovery] Peer addresses from bootnode:", nodeAddress)
	go blockchain.StartBlockchainProcessor()
	go blockchain.StartBlockCreator()
	p2p.StartClient(nodeAddress)
}
