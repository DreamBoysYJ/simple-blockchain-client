// utils.go

package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net"

	"golang.org/x/crypto/sha3"
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

// String => BigInt 타입으로 변경
func ConvertStringToBigInt(value string) (*big.Int, error) {
	bigIntValue := new(big.Int)

	// 10진수 문자열을 *big.Int로 변환
	_, success := bigIntValue.SetString(value, 10)
	if !success {
		return nil, fmt.Errorf("문자열을 big.Int로 변환하는 데 실패했습니다: %s", value)
	}

	return bigIntValue, nil
}

func Keccak256(data []byte) []byte {
	// hash 함수 생성
	hash := sha3.NewLegacyKeccak256()
	// hash 함수에 데이터 입력
	hash.Write(data)
	// 해시값을 계산하여 반환
	return hash.Sum(nil)

}

func BytesToHex(data []byte) string {
	return hex.EncodeToString(data)
}

func Keccak256Hex(data []byte) string {
	hash := Keccak256(data)
	return hex.EncodeToString(hash)
}
