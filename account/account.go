package account

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"

	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"

	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

func PublicKeyToAddress(pubKey []byte) (string, error) {
	// pubKey는 압축되지 않은 공개키여야 함(len = 65, 첫 바이트 0x04)

	if len(pubKey) != 65 {
		return "", fmt.Errorf("invalid public key length, must be 65 bytes, but got %d bytes", len(pubKey))
	}

	if pubKey[0] != 0x04 {
		return "", fmt.Errorf("invalid public key format, must start with 0x04 for uncompressed keys, but got %v", pubKey[0])
	}

	// keccak256 계산
	hash := utils.Keccak256(pubKey[1:])

	// 마지막 20바이트를 이더리움 주소로 사용
	address := hash[len(hash)-20:]
	// 16진수로 출력하고 0x 붙이기
	return fmt.Sprintf("0x%x", address), nil
}

func StoreAccount(address string) (bool, error) {
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

func StoreAccountForGenesisMiner(address string, initialBalance *big.Int) (bool, error) {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		// 에러
		return false, fmt.Errorf("db 접근 불가 : %v", err)
	}
	key := append([]byte("account:"), []byte(address)...)

	account := &Account{
		Balance: initialBalance,
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

func CreateAccount() (string, string, error) {

	// 1. 개인키 생성
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("개인키 생성 실패 : %v", err)
	}

	// 2. 개인키 16진수 문자열로 변환
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hexutil.Encode(privateKeyBytes)[2:] // 0x 제거

	// 3. 공개키 생성
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	publicKeyBytes := crypto.FromECDSAPub(publicKey)
	//publicKeyHex := hexutil.Encode(publicKeyBytes)[4:] // 0x04 제거

	// 4. 개인키, 공개키

	address, err := PublicKeyToAddress(publicKeyBytes)
	if err != nil {
		return "", "", fmt.Errorf("주소 생성 실패 : %v", err)
	}

	return privateKeyHex, address, nil

}

func IsValidAddress(address string) bool {
	// 0x 시작 && 총 42자 && 40자 16진수
	// ^ : 시작
	// [0-9a-fA-f] : 이 부분은 괄호 안에 있는 문자들 중 하나를 허용
	// {40} 앞 패턴이 몇번 있어야 하는지
	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}

func PublicKeyToBytes(pubKey *secp256k1.PublicKey) []byte {
	return pubKey.SerializeUncompressed()
}

func CheckAccountState(from, to, value string, nonce uint64) error {
	// 1. from 계정이 존재하는지 확인
	fromExists, err := AccountExists(from)
	if err != nil {
		return fmt.Errorf("failed to check if `from` account exists : %v", err)
	}
	if !fromExists {
		return fmt.Errorf("from account must already exist")
	}

	// 2. from의 value 확인
	fromAccount, err := GetAccount(from)
	if err != nil {
		return fmt.Errorf("failed to retrieve `from` account: %v", err)
	}
	valueBigInt, err := utils.ConvertStringToBigInt(value)
	if err != nil {
		return fmt.Errorf("failed to convert value to big.Int: %v", err)
	}

	if fromAccount.Balance.Cmp(valueBigInt) < 0 {
		return fmt.Errorf("insufficient funds : available balance is %d", fromAccount.Balance)
	}

	// 3. from의 nonce 확인
	if fromAccount.Nonce > nonce {
		return fmt.Errorf("nonce mismatch : expected %d, got %d", fromAccount.Nonce, nonce)
	}

	// 4. to 계정이 없다면 생성해주기
	toExists, err := AccountExists(to)
	if err != nil {
		return fmt.Errorf("failed to check if `to` account exists : %v", err)
	}
	if !toExists {
		_, err := StoreAccount(to)
		if err != nil {
			return fmt.Errorf("failed to create `to` account : %v", err)
		}
	}
	return nil

}

// 개인 키 로드 함수
func LoadPrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}
	return privateKey, nil
}

// 메시지 서명 함수
func SignMessage(hash []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %v", err)
	}
	return hex.EncodeToString(signature), nil
}
