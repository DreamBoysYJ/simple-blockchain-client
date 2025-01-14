package blockchain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"simple_p2p_client/account"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	db "github.com/syndtr/goleveldb/leveldb"
)

type RawTransaction struct {
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     *big.Int `json:"value"`
	Nonce     uint64   `json:"nonce"`
	Signature string   `json:"signature"`
}

type Transaction struct {
	Hash      string   `json:"hash"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Value     *big.Int `json:"value"`
	Nonce     uint64   `json:"nonce"`
	Signature string   `json:"signature"`
}

// 서명 검증
func VerifySignature(messageHash []byte, signature []byte, fromAddress string) (bool, error) {

	// 공개키 복구
	pubKey, err := crypto.Ecrecover(messageHash, signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key : %v", err)
	}

	// 공개키를 바이트 배열로 변환 (압축되지 않은 형식)
	// fmt.Printf("Recovered public key (uncompressed): %x\n", pubKey)

	// 복구된 공개키로부터 주소 생성
	address, err := account.PublicKeyToAddress(pubKey)
	// fmt.Printf("Address: %s\n", address)

	if err != nil {
		return false, fmt.Errorf("failed to create address from public key : %v", err)
	}

	// from과 비교
	if strings.EqualFold(address, fromAddress) {
		return true, nil
	} else {
		return false, nil
	}

}

// 트랜잭션 필드들 유효성 검증 (주소 양식, 빈 값, value 크기)
func ValidateTransactionFields(from, to, value, signature string, nonce uint64) error {

	// 1. 빈 인자 없는지 확인
	if from == "" || to == "" || value == "" || signature == "" {
		return fmt.Errorf("missing required fields: 'from', 'to', 'value', 'signature'")
	}

	// 2. value > 0 인지
	valueInt, err := strconv.Atoi(value)
	if err != nil || valueInt <= 0 {
		return fmt.Errorf("invalid value : must be a positive integer")
	}

	// TODO : 삭제하기, 어차피 uint값이라 무조건 통과됨.
	// 3. nonce >=0
	if nonce < 0 {
		return fmt.Errorf("inavlid nonce : must be a non-negative integer")
	}

	// 4. from, to 주소 양식이 올바른지
	if !account.IsValidAddress(from) {
		return fmt.Errorf("invalid address : address 'from' format is wrong")
	}
	if !account.IsValidAddress(to) {
		return fmt.Errorf("invalid address : address 'to' format is wrong")
	}

	return nil

}

// 트랜잭션 각 필드들을 조합해 트랜잭션 구조체, json 형태 반환
func CreateTransaction(from, to, signature string, value *big.Int, nonce uint64) (Transaction, string, error) {
	// 1. RawTransaction 생성
	rawTransaction := RawTransaction{
		From:      from,
		To:        to,
		Value:     value,
		Nonce:     nonce,
		Signature: signature,
	}

	// 2. JSON 직렬화
	jsonRawTransaction, err := json.Marshal(rawTransaction)
	if err != nil {
		return Transaction{}, "", fmt.Errorf("failed to encode raw transaction to JSON: %v", err)

	}

	// 3. 해시 생성
	jsonRawTransactionHash := utils.Keccak256(jsonRawTransaction)
	jsoonRawTransactionHashStr := fmt.Sprintf("0x%s", utils.BytesToHex(jsonRawTransactionHash))

	// 4. 트랜잭션 생성
	fullTransaction := Transaction{
		Hash:      jsoonRawTransactionHashStr,
		From:      from,
		To:        to,
		Value:     value,
		Nonce:     nonce,
		Signature: signature,
	}

	return fullTransaction, string(jsonRawTransaction), nil
}

// 트랜잭션 유효성 검증, 멤풀에 저장, Json 트랜잭션 반환
func ProcessTransaction(rawTransactionMessage string) (string, error) {

	// 1. RawTransaction 구조체로 변환
	var rawTransaction RawTransaction
	err := json.Unmarshal([]byte(rawTransactionMessage), &rawTransaction)
	if err != nil {
		return "", fmt.Errorf("failed to parse raw transaction: %v", err)
	}

	// 2. 트랜잭션 필드 검증
	err = ValidateTransactionFields(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Signature, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction field validation failed: %v", err)
	}

	// 3. 서명 검증
	message := fmt.Sprintf("%s%s%s%d", rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
	messageHash := utils.Keccak256([]byte(message))
	decodedSignature, err := hex.DecodeString(rawTransaction.Signature)
	if err != nil {
		return "", fmt.Errorf("invalid signature format: %v", err)
	}
	isValidSig, err := VerifySignature(messageHash, decodedSignature, rawTransaction.From)
	if err != nil || !isValidSig {
		return "", fmt.Errorf("signature verification failed: %v", err)
	}

	// 4. 계정 상태 확인
	err = account.CheckAccountState(rawTransaction.From, rawTransaction.To, rawTransaction.Value.String(), rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("account state validation failed: %v", err)
	}

	// 5. 트랜잭션 생성
	tx, jsonRawTransactionStr, err := CreateTransaction(rawTransaction.From, rawTransaction.To, rawTransaction.Signature, rawTransaction.Value, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %v", err)
	}

	// 6. Mempool에 저장
	err = defaultMempool.AddTransaction(tx, rawTransaction.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to append transaction to mempool: %v", err)
	}

	fmt.Printf("[TX] Validation completes, Hash : %v", tx.Hash)

	// 7. 반환
	return jsonRawTransactionStr, nil
}

// 트랜잭션 실행(db 상태 변경)
func ExecuteTransactions(transactions []Transaction) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		return fmt.Errorf("failed to get dbinstance : %v", err)
	}

	for _, tx := range transactions {

		batch := new(db.Batch)

		// 1. From 가져오기
		fromAccount, err := account.GetAccount(tx.From)
		if err != nil {
			return fmt.Errorf("failed to retrieve from account : %s : %v", tx.From, err)
		}

		// 2. To 가져오기 (없는 경우 생성)
		toAccount, err := account.GetAccount(tx.To)
		if err != nil {
			if err == db.ErrNotFound { // 없는 경우 생성
				_, createErr := account.StoreAccount(tx.To)
				if createErr != nil {
					return fmt.Errorf("failed to create to account %s: %v", tx.To, err)
				}
				toAccount, _ = account.GetAccount(tx.To)
			} else {
				return fmt.Errorf("failed to retrieve to account %s:%v", tx.To, err)
			}
		}

		// 3. 잔액 검증 및 트랜잭션 실행
		txValue := tx.Value
		if fromAccount.Balance.Cmp(txValue) < 0 {
			return fmt.Errorf("insufficient balance in from account %s", tx.From)
		}

		fromAccount.Balance.Sub(fromAccount.Balance, txValue)
		toAccount.Balance.Add(toAccount.Balance, txValue)
		fromAccount.Nonce++

		// 4. 계정 데이터 저장 (배치 작업 추가)
		fromAccountKey := append([]byte("account:"), []byte(tx.From)...)
		toAccountKey := append([]byte("account:"), []byte(tx.To)...)

		fromAccountJSON, err := json.Marshal(fromAccount)
		if err != nil {
			return fmt.Errorf("failed to serialize from account: %v", err)
		}

		toAccountJSON, err := json.Marshal(toAccount)
		if err != nil {
			return fmt.Errorf("failed to serialize to account: %v", err)
		}

		batch.Put(fromAccountKey, fromAccountJSON)
		batch.Put(toAccountKey, toAccountJSON)

		// 5. 배치 실행
		err = dbInstance.Write(batch, nil)
		if err != nil {
			return fmt.Errorf("failed to execute batch write : %v", err)
		}

		fmt.Printf("[TX] Execution completed, Hash : %s, From : %s, To: %s, Value : %s\n", tx.Hash, tx.From, tx.To, tx.Value)

	}

	return nil
}

// 블록 내 트랜잭션들 검증
func ProcessTransactionFromBlock(tx Transaction) (string, error) {
	// 1. 트랜잭션 필드 검증
	err := ValidateTransactionFields(tx.From, tx.To, tx.Value.String(), tx.Signature, tx.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction field validation failed : %v", err)
	}

	// 2. 서명 검증
	txMessage := fmt.Sprintf("%s%s%s%d", tx.From, tx.To, tx.Value.String(), tx.Nonce)
	txMessageHash := utils.Keccak256([]byte(txMessage))
	decodedSignature, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return "", fmt.Errorf("invalid transaction signature format : %v", err)
	}
	isValidSig, err := VerifySignature(txMessageHash, decodedSignature, tx.From)
	if err != nil || !isValidSig {
		return "", fmt.Errorf("transaction signature validation failed : %v", err)
	}

	// 3. 계정 상태 확인
	err = account.CheckAccountState(tx.From, tx.To, tx.Value.String(), tx.Nonce)
	if err != nil {
		return "", fmt.Errorf("transaction state validation failed : %v", err)
	}

	fmt.Printf("[TX] Validation compeleted, hash : %v\n", tx.Hash)

	// 4. 성공적으로 검증된 트랜잭션 반환
	return tx.Hash, nil
}
