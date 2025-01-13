package rpcserver

import (
	"fmt"
	"net/http"
	"simple_p2p_client/account"
)

type AccountAPI struct{}

type GetAccountArgs struct {
	Address string `json:"address"`
}

type GetAccountReply struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

// 주소 조회
func (s *AccountAPI) GetAccountInfo(r *http.Request, args *GetAccountArgs, reply *GetAccountReply) error {
	// 입력된 주소 유효성 검사
	if !account.IsValidAddress(args.Address) {
		return fmt.Errorf("invalid address format: %s", args.Address)
	}

	// 계정 정보 조회
	accountData, err := account.GetAccount(args.Address)
	if err != nil {
		return fmt.Errorf("failed to retrieve account info: %v", err)
	}

	// 응답 데이터 설정
	reply.Address = args.Address
	reply.Balance = accountData.Balance.String() // big.Int -> 문자열 변환
	reply.Nonce = accountData.Nonce

	return nil
}

type NewAccountArgs struct {
}

type NewAccountReply struct {
	PrivateKey string
	Address    string
}

// 랜덤으로 새 계정 생성 (개인키, 주소 반환)
func (s *AccountAPI) NewAccount(r *http.Request, args *NewAccountArgs, reply *NewAccountReply) error {

	// 랜덤 계정 생성
	privateKey, address, err := account.CreateAccount()
	if err != nil {
		return fmt.Errorf("making new account error : %v", err)
	}

	// 주소, 개인키 반환
	reply.Address = address
	reply.PrivateKey = privateKey
	return nil
}
