package leveldb

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestTestLevelDbOne(t *testing.T) {
	getFromtheEnd()

}
func TestTestLevelDbTwo(t *testing.T) {

	storeLastblock()
}

// 1. 뒤부터 순회
func getFromtheEnd() {

	// 데이터베이스 열기 (없으면 생성)
	// path : 프로젝트 실행 pwd
	db, err := leveldb.OpenFile("test", nil)
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll("test")
	defer db.Close()

	// 30만개 블록 저장
	for i := 1; i <= 300000; i++ {
		key := fmt.Sprintf("block:%d", i) //  문자열을 서식을 지정해 생성
		value := strconv.Itoa(i)          // 숫자 => 문자열
		err = db.Put([]byte(key), []byte(value), nil)
		if err != nil {
			log.Fatalf("Failed to put block %d: %v", i, err)

		}
	}

	// 뒤부터 순회
	lastBlock, err := getLastBlockUsingIterator(db)
	if err != nil {
		log.Fatalf("Failed to get last block : %v", err)
	}
	fmt.Printf("LAST BLOCK :::: %s", lastBlock)

}

// 2. lastblock 따로 저장
func storeLastblock() {
	db, err := leveldb.OpenFile("test2", nil)
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll("test2")
	defer db.Close()

	// 30만개 블록 저장
	// 30만개 last block 저장
	for i := 1; i <= 300000; i++ {
		key := fmt.Sprintf("block:%d", i)
		value := strconv.Itoa(i)
		err = db.Put([]byte(key), []byte(value), nil)
		if err != nil {
			log.Fatalf("Failed to put block %d: %v", i, err)
		}
		err = db.Put([]byte("lastblock"), []byte(value), nil)
		if err != nil {
			log.Fatalf("Failed to put lastblock : %d: %v", i, err)
		}
	}

	// lastblock 조회
	lastBlock, err := GetLastBlock(db)
	if err != nil {
		log.Fatalf("Failed to get lastblock : %v", err)
	}
	fmt.Printf("2번째 테스트 ::: %s", lastBlock)

}

func getLastBlockUsingIterator(db *leveldb.DB) ([]byte, error) {
	var lastBlock []byte

	iter := db.NewIterator(nil, nil)
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := iter.Key()

		// block:으로 시작하는 Key 찾으면 해당 블록이 마지막 블록
		if len(key) > 6 && string(key[:6]) == "block:" {
			lastBlock = iter.Value()
			iter.Release()
			return lastBlock, nil
		}
	}
	iter.Release()

	// 순회 중 에러 발생시 반환
	if err := iter.Error(); err != nil {
		return nil, err
	}

	// 블록 찾지 못한 경우
	return nil, fmt.Errorf("no blocks found")
}
