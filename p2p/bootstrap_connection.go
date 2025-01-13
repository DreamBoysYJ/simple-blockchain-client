// bootstrap_connection.go

package p2p

import (
	"bufio"
	"fmt"
	"io"
	"net"
	pc "simple_p2p_client/protocol_constants"
	"simple_p2p_client/utils"
	"strings"
	"time"
)

// 글로벌 변수 : 모든 피어의 연결 정보 저장하는 리스트
var ConnectedPeers []net.Conn

// Bootstrap : UDP version
func ConnectBootstrapNode(bootstrapAddress string, udpServerAddress string) ([]string, error) {
	var nodeLists []string

	// create UDP address, socket
	// net.ResolveUDPAddr(network,address) : UDP 주소 객체 변환
	bootstrapUDPAddr, err := net.ResolveUDPAddr("udp", bootstrapAddress)
	if err != nil {
		return nodeLists, fmt.Errorf("error resolving UDP addres : %v", err)
	}

	// DialUDP : 로컬 소켓 생성 (TCP는 실제 연결까지 진행)
	conn, err := net.DialUDP("udp", nil, bootstrapUDPAddr)
	if err != nil {
		return nodeLists, fmt.Errorf("error connecting to bootstrap node : %v", err)
	}
	defer conn.Close()

	// 1. Send Ping
	fmt.Println("[Node Discovery] Sending 'Ping' to bootnode")
	pingMessage := []byte{pc.NodeDiscoveryPing}
	_, err = conn.Write(pingMessage)
	if err != nil {
		return nodeLists, fmt.Errorf("error sending Ping message : %v", err)
	}

	// 2. Waiting Pong
	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Read에 대해 최대 기다릴 수 있는 시간
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return nodeLists, fmt.Errorf("error receiving 'Pong' : %v", err)
	}

	// 3. Send FindNode
	if buffer[0] == pc.NodeDiscoveryPong {
		fmt.Println("[Node Discovery] Received 'Pong' from bootnode")
		fmt.Println("[Node Discovery] Sending 'FindNode' to bootnode")
		findNodeMessage := append([]byte{pc.NodeDiscoveryFindNode}, []byte(udpServerAddress)...)
		_, err = conn.Write(findNodeMessage)
		if err != nil {
			return nodeLists, fmt.Errorf("error sending FindNode message : %v", err)
		}

		// 4. Waiting Neighbors
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, _, err = conn.ReadFromUDP(buffer)
		if err != nil {
			return nodeLists, fmt.Errorf("error receiving Neighbors message : %v", err)
		}

		if buffer[0] == pc.NodeDiscoveryNeighbors {
			message := string(buffer[1:n])
			nodeLists = strings.Split(strings.TrimSpace(message), ",")
			// fmt.Println("부트스트랩 노드로부터 받은 노드들 : ", nodeLists)
			return nodeLists, nil

		} else {
			return nodeLists, fmt.Errorf("did not receive Neighbors message")
		}

	} else {
		return nodeLists, fmt.Errorf("did not receive Pong message")
	}

}

// TO BE DEPRECATED : 테스트용으로 TCP 연결도 생성, 추후 삭제 예정
// Bootstrap : TCP version
func ConnectBootstrapNodeTcp(bootstrapAddress string, serverAddress string) []string {

	var connectedNodes []string

	// 부트스트랩 노드에 연결
	conn, err := net.Dial("tcp", bootstrapAddress)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error connecting to bootstrap node: %v", err))
		return connectedNodes
	}
	defer conn.Close()

	// 내 서버 주소를 부트스트랩 노드에 전달
	fmt.Println("Sending server address to bootstrap node : ", serverAddress)

	const Neighbors = 0x01
	protocol := []byte{Neighbors}
	myAddress := []byte(serverAddress)
	protocol = append(protocol, myAddress...)

	_, err = conn.Write(protocol)
	// _, err = conn.Write([]byte(serverAddress + "\n"))
	if err != nil {
		utils.PrintError(fmt.Sprintf("Error sending server address: %v", err))
	}

	// 부트스트랩 노드로부터 다른 노드들 주소 받기
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		if err == io.EOF {
			utils.PrintError("Bootstrap node has closed the connection.")

		} else {
			utils.PrintError(fmt.Sprintf("Error reading from bootstrap node: %v", err))
		}
		return connectedNodes
	}

	// 받은 메시지를 자료 구조에 저장
	connectedNodes = strings.Split(strings.TrimSpace(message), ",")
	fmt.Println("부트스트랩 노드로 부터 받은 노드들: ", connectedNodes)
	return connectedNodes

}
