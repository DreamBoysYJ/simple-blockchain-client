package account

import (
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/leveldb"

	db "github.com/syndtr/goleveldb/leveldb"
)

type Account struct {
	Balance *big.Int
	Nonce   uint64
}

func GetAccount(address string) (*Account, error) {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return nil, fmt.Errorf("db 접근 불가 : %v", err)
	}

	accountKey := append([]byte("account:"), []byte(address)...)

	accountValue, err := dbInstance.Get([]byte(accountKey), nil)

	if err != nil {
		return nil, fmt.Errorf("해당 키 없음 : %v", err)
	}

	var account Account

	err = json.Unmarshal(accountValue, &account)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal failed : %v", err)
	}

	return &account, nil

}

func AccountExists(address string) (bool, error) {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		// 에러
		return false, fmt.Errorf("db 접근 불가 : %v", err)
	}
	key := append([]byte("account:"), []byte(address)...)

	_, err = dbInstance.Get(key, nil)
	if err != nil {
		// 해당 키가 없는 경우
		if err == db.ErrNotFound {
			return false, nil
		}
		// 에러
		return false, fmt.Errorf("db 오류 발생 : %v", err)
	}

	// 존재
	return true, nil

}

func CreateAccount(address string) (bool, error) {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		// 에러
		return false, fmt.Errorf("db 접근 불가 : %v", err)
	}
	key := append([]byte("account:"), []byte(address)...)

	account := &Account{
		Balance: big.NewInt(0),
		Nonce:   0,
	}

	accountJson, err := json.Marshal(account)
	if err != nil {
		return false, fmt.Errorf("json Marshal failed : %v", err)
	}

	err = dbInstance.Put(key, accountJson, nil)
	if err != nil {
		return false, fmt.Errorf("계정 저장 실패 : %v", err)
	}

	return true, nil

}
