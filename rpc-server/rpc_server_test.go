package rpcserver

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestCreateMessage(t *testing.T) {

	// 개인키
	privateKeyHex := "7ac125dda168b44ee9fc0d8db3a804ef86b3cc50206a0112b25373d622cf78f7"
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v\n", err)
	}
	fmt.Printf("PRIVATE KEY :::%v", privateKey)

	// 공개키 추출
	publicKey := privateKey.Public().(*ecdsa.PublicKey)

	// 공개키의 X, Y 좌표 출력
	publicKeyBytes := crypto.FromECDSAPub(publicKey)
	fmt.Printf("Uncompressed Public Key: %x\n", publicKeyBytes)

	// 서명할 메시지 (메시지 해시가 아닌 원문)
	message := []byte("0xde589C867174C349d00e9b582867aF5c13A746790x7a227D5902cA52C0C3C61304533bfF4632Fce1451001")

	// 메시지 Keccak256 해시
	messageHash := crypto.Keccak256Hash(message)

	// 메시지 서명
	signature, err := crypto.Sign(messageHash.Bytes(), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign message: %v", err)
	}

	// 서명 결과 출력 (R, S, V 값 포함)
	fmt.Printf("Signature: %s\n", hex.EncodeToString(signature))

	// R, S, V 값 분리
	r := signature[:32]
	s := signature[32:64]
	v := signature[64:]

	fmt.Printf("R: %x\n", r)
	fmt.Printf("S: %x\n", s)
	fmt.Printf("V: %x\n", v)
}