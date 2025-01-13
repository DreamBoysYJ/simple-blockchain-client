package leveldb

import (
	"fmt"
	"os"

	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

var (
	dbInstance *leveldb.DB
	once       sync.Once
	dbPath     = "leveldb/database"
)

// // calculates the absolute path for the database
// func GetDBPath(nodeID string) string {
// 	// Get the user's home directory
// 	homeDir, err := os.UserHomeDir()
// 	if err != nil {
// 		panic(fmt.Sprintf("Failed to get home directory: %v", err))
// 	}
// 	// Construct the absolute path for the database
// 	return filepath.Join(homeDir, ".simple_p2p_client", "db", nodeID)
// }

func SetDBPath(path string) {
	dbPath = path
}

// db 인스턴스 : db와의 연결을 관리하는 객체, 구조체
func GetDBInstance() (*leveldb.DB, error) {
	var err error
	once.Do(func() {
		dbInstance, err = leveldb.OpenFile(dbPath, nil)
	})
	return dbInstance, err
}

func IsDBOpened() bool {
	return dbInstance != nil
}

func CloseDB() error {
	if dbInstance != nil {
		return dbInstance.Close()
	}
	return nil
}

// DB 디렉토리를 확인하고 없으면 생성
func ensureDBDirectory() error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		err := os.MkdirAll(dbPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create DB directory: %v", err)
		}
	}
	return nil
}

// Init LevelDB instance
func InitDB() error {
	if err := ensureDBDirectory(); err != nil {
		return err
	}
	_, err := GetDBInstance()
	return err
}

// LevelDB instance 종료
func CleanupDB() error {
	if IsDBOpened() {
		return CloseDB()
	}
	return nil
}

func GetLastBlock(db *leveldb.DB) ([]byte, error) {

	lastBlockValue, err := db.Get([]byte("lastblock"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get lastblock value : %v", err)
	}

	return lastBlockValue, nil
}
