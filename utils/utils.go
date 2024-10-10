// utils.go

package utils

import (
	"fmt"
	"net"
)

// 메시지를 출력하고 입력 프롬프트를 다시 출력
func PrintMessage(message string) {

	// 현재 줄 지우고 메시지 출력
	fmt.Printf("\r\033[K%s\n", message)

	// 입력 프롬프트 다시 출력
	fmt.Print("Enter message : ")
}

// 에러 메시지를 출력하고 입력 프롬프트를 다시 출력
func PrintError(errorMessage string) {

	// 현재 줄 지우고 에러 메시지 출력
	fmt.Printf("\r\033[K[ERROR] %s\n", errorMessage)

	// 입력 프롬프트 다시 출력
	fmt.Print("Enter message : ")
}

// 슬라이스에서 i번째 요소를 제거하는 코드
func RemoveConn(peers []net.Conn, target net.Conn) []net.Conn {
	for i, conn := range peers {
		if conn == target {
			// 해당 인덱스의 요소를 제거
			return append(peers[:i], peers[i+1:]...)
		}
	}
	return peers // 값이 없으면 원래 슬라이스 반환
}
