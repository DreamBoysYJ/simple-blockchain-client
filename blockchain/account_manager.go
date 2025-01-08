package blockchain

import (
	"fmt"
	"simple_p2p_client/account"
	"simple_p2p_client/leveldb"
)

var NodeAccount string // 프로그램을 실행하는 노드의 주소

// 노드 계정을 초기화하는 함수
func InitializeNodeAccount() error {
	// DB 접근
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("DB 접근 실패 : %v", err)
	}

	// 기존 노드 계정 로드
	nodeAccountKey := "nodeAccount"
	accountData, err := dbInstance.Get([]byte(nodeAccountKey), nil)
	if err == nil {
		// 기존 계정 있으면 불러오기
		NodeAccount = string(accountData)
		fmt.Printf("[ACCOUNT] Node Account load complete : %s\n", NodeAccount)
		return nil
	}

	// 없으면 생성
	_, address, err := account.CreateAccount()
	if err != nil {
		return fmt.Errorf("노드 게정 생성 실패 : %v", err)
	}

	// 새 계정 저장
	err = dbInstance.Put([]byte(nodeAccountKey), []byte(address), nil)
	if err != nil {
		return fmt.Errorf("노드 계정 저장 실패 : %v", err)
	}

	NodeAccount = address
	fmt.Printf("새로운 노드 계정 생성 : %s\n", NodeAccount)
	return nil
}
