// client.g

package p2p

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"simple_p2p_client/mediator"
	pc "simple_p2p_client/protocol_constants"
	"simple_p2p_client/utils"
	"strings"
	"time"
)

var messageChannel = make(chan string) // 메시지 송신용 채널

func StartClient(nodeAddress []string) {

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
				utils.PrintError(fmt.Sprintf("Error resolving UDP address : %v", err))
				continue
			}
			conn, err := net.DialUDP("udp", nil, nodeUDPAddr)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error connecting to node : %v", err))
				continue
			}

			// 1. Send Ping
			fmt.Printf("[Node Discovery] Sending 'Ping' to node : %v\n", nodeUDPAddr)
			pingMessage := []byte{pc.NodeDiscoveryPing}
			_, err = conn.Write(pingMessage)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error sending Ping message : %v", err))
				continue
			}

			// 2. Waiting Pong
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error receiving Pong message : %v", err))
				continue
			}
			// 3. Send ENRRequest
			if buffer[0] == pc.NodeDiscoveryPong {
				fmt.Printf("[Node Discovery] Received 'Pong' from node : %v\n", nodeUDPAddr)

				fmt.Printf("[Node Discovery] Sending 'ENRRequest' from node : %v\n", nodeUDPAddr)

				_, err = conn.Write([]byte{pc.NodeDiscoveryENRRequest})
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error sending ENRRequest : %v", err))
					continue
				}

				// 4. Waiting ENRResponse
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				n, _, err = conn.ReadFromUDP(buffer)
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error receiving ENRResponse : %v", err))
					continue
				}

				if buffer[0] == pc.NodeDiscoveryENRResponse {
					fmt.Printf("[Node Discovery] Received 'ENRResponse' from node : %v\n", nodeUDPAddr)
					fmt.Printf("[Node Discovery] TCP server : %v\n", nodeUDPAddr)

					tcpServer := string(buffer[1:n])
					fmt.Printf("[Node Discovery] TCP server : %v, dialing...\n", tcpServer)

					// 5. TCP 연결
					conn, err := net.Dial("tcp", tcpServer)
					if err != nil {
						utils.PrintError(fmt.Sprintf("Error reading from connection : %v", err))

					} else {
						fmt.Printf("[Node Discovery] Successfully connected to node,  %v\n", tcpServer)
						ConnectedPeers = append(ConnectedPeers, conn)
					}

				}

			}

		}
	} else {
		fmt.Println("No node addresses provided. Waiting for connection...")
	}

	// 각 피어와 병렬로 메시지 받기 고루틴
	for _, conn := range ConnectedPeers {
		go RefactorHandleIncomingMessages(conn) // 각 연결에 대해 별도의 고루틴으로 처리
	}

	// 메시지 입력 고루틴
	go func() {
		for {
			fmt.Println("Enter Message : ")
			message, _ := reader.ReadString('\n')
			message = strings.TrimSpace(message)
			messageChannel <- message
		}
	}()

	// 메시지 전송 고루틴
	go func() {
		for message := range messageChannel {
			HandleSendingMessages(ConnectedPeers, 0x00, message)
		}
	}()

	// BlockchainToP2P 채널에서 메시지를 읽어와 피어들에게 전송
	go func() {
		for processedMessage := range mediator.GetMediatorInstance().BlockchainToP2P {

			if len(processedMessage) == 0 {
				utils.PrintError("Received empty message, skipping...")
				continue
			}

			// 첫 바이트 (프로토콜 ID)와 나머지 메시지 분리
			protocolID := processedMessage[0]
			messageContent := processedMessage[1:]

			utils.PrintMessage(fmt.Sprintf("[P2P] Forwarding message to peers: %s", processedMessage))
			HandleSendingMessages(ConnectedPeers, protocolID, messageContent)
		}
	}()

	select {} // 메인 루프를 멈추지 않기 위해 블로킹 처리

}

// 메시지 보내기 함수
func HandleSendingMessages(peers []net.Conn, protocolID byte, message string) {

	fullMessage := append([]byte{protocolID}, []byte(message+"\n")...)

	for _, conn := range peers {
		if conn != nil {
			// 메시지 전송
			_, err := conn.Write([]byte(fullMessage))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error sending message to peer : %v", err))
				ConnectedPeers = utils.RemoveConn(ConnectedPeers, conn)
				conn.Close()
			}
		}
	}
}
