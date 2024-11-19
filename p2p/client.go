// client.g

package p2p

import (
	"bufio"
	"fmt"
	"net"
	"os"
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
				return
			}
			conn, err := net.DialUDP("udp", nil, nodeUDPAddr)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error connecting to node : %v", err))
				return
			}

			// 1. Send Ping
			fmt.Println("Sending Ping to node")
			pingMessage := []byte{pc.NodeDiscoveryPing}
			_, err = conn.Write(pingMessage)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error sending Ping message : %v", err))
				return
			}

			// 2. Waiting Pong
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error receiving Pong message : %v", err))
				return
			}
			// 3. Send ENRRequest
			if buffer[0] == pc.NodeDiscoveryPong {
				fmt.Println("Received Pong message")
				fmt.Println("Sending ENRRequest")
				_, err = conn.Write([]byte{pc.NodeDiscoveryENRRequest})
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error sending ENRRequest : %v", err))
					return
				}

				// 4. Waiting ENRResponse
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				n, _, err = conn.ReadFromUDP(buffer)
				if err != nil {
					utils.PrintError(fmt.Sprintf("Error receiving ENRResponse : %v", err))
					return
				}

				if buffer[0] == pc.NodeDiscoveryENRResponse {
					tcpServer := string(buffer[1:n])
					fmt.Println("TCP SERVER :::", tcpServer)

					// 5. TCP 연결
					conn, err := net.Dial("tcp", tcpServer)
					if err != nil {
						utils.PrintError(fmt.Sprintf("Error reading from connection : %v", err))

					} else {
						fmt.Println("Successfully connected to", tcpServer)
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
		go HandleIncomingMessages(conn) // 각 연결에 대해 별도의 고루틴으로 처리
	}

	// 메시지 입력 고루틴
	go func() {
		for {
			fmt.Print("Enter Message : ")
			message, _ := reader.ReadString('\n')
			message = strings.TrimSpace(message)
			messageChannel <- message
		}
	}()

	// 메시지 전송 고루틴
	go func() {
		for message := range messageChannel {
			HandleSendingMessages(ConnectedPeers, message)
		}
	}()

	select {} // 메인 루프를 멈추지 않기 위해 블로킹 처리

}

// 메시지 보내기 함수
func HandleSendingMessages(peers []net.Conn, message string) {
	for _, conn := range peers {
		if conn != nil {
			_, err := conn.Write([]byte(message + "\n"))
			if err != nil {
				utils.PrintError(fmt.Sprintf("Error sending message to peer : %v", err))
				ConnectedPeers = utils.RemoveConn(ConnectedPeers, conn)
				conn.Close()
			}
		}
	}
}
