package rpcserver

import (
	encodingJson "encoding/json"
	"fmt"
	"net/http"
	"simple_p2p_client/account"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/mediator"
	"simple_p2p_client/protocol_constants"
	"simple_p2p_client/utils"
)

type TransactionAPI struct{}

type SendTransactionArgs struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
}

type SendTransactionReply struct {
	TxHash string `json:"txHash"`
}

// 서명한 트랜잭션 제출
func (s *TransactionAPI) SendTransaction(r *http.Request, args *SendTransactionArgs, reply *SendTransactionReply) error {

	// 1. 트랜잭션 데이터를 JSON 형식으로 포맷

	valueBigInt, err := utils.ConvertStringToBigInt(args.Value)
	if err != nil {
		return fmt.Errorf("failed to convert value to big.int: %v", err)
	}

	rawTransaction := blockchain.RawTransaction{
		From:      args.From,
		To:        args.To,
		Value:     valueBigInt,
		Nonce:     args.Nonce,
		Signature: args.Signature,
	}

	rawTransactionBytes, err := encodingJson.Marshal(rawTransaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %v", err)
	}

	// 2. 트랜잭션 메시지를 채널로 전달

	mediatorInstance := mediator.GetMediatorInstance()
	mediatorInstance.P2PToBlockchain <- fmt.Sprintf("%c%s", protocol_constants.P2PTransactionMessage, string(rawTransactionBytes))

	// 3. 트랜잭션 해시값 생성 및 반환
	transactionHash := utils.Keccak256Hex(rawTransactionBytes)
	reply.TxHash = transactionHash

	// fmt.Println("Transaction sent to mediator. Hash:", transactionHash)
	return nil

}

type SendRawTransactionArgs struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Nonce uint64 `json:"nonce"`
}

type SendRawTransactionReply struct {
	Signature string `json:"signature"`
}

// TO BE DEPRECATED : 특정 주소의 서명값만 반환 가능. 테스트 용도, 이후 삭제
// 서명되지 않은 트랜잭션 전달(서명 반환)
func (s *TransactionAPI) SendRawTransaction(r *http.Request, args *SendRawTransactionArgs, reply *SendRawTransactionReply) error {

	// 1. 메시지 생성 (from, to, value, nonce를 연결)
	message := fmt.Sprintf("%s%s%s%d", args.From, args.To, args.Value, args.Nonce)

	// 2. 메시지 해싱 (Keccak256)
	messageHash := utils.Keccak256([]byte(message))

	// 3. 개인 키 로드 (개인 키를 안전하게 관리해야 함)
	privateKeyHex := "7ac125dda168b44ee9fc0d8db3a804ef86b3cc50206a0112b25373d622cf78f7" // 실제 서비스에서는 안전한 방식으로 관리 필요
	privateKey, err := account.LoadPrivateKey(privateKeyHex)                            // account 패키지에서 개인 키 로드 함수 구현 필요
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}

	// 4. 메시지 서명
	signature, err := account.SignMessage(messageHash, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	// 5. 서명 반환
	reply.Signature = signature
	return nil
}
