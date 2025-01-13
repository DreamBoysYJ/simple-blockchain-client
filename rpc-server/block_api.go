package rpcserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"simple_p2p_client/blockchain"
	"simple_p2p_client/leveldb"
	"simple_p2p_client/utils"
)

type BlockAPI struct{}

type BlockNumberArgs struct{}
type BlockNumberReply struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// 마지막 블록의 길이 조회
func (b *BlockAPI) GetBlockNumber(r *http.Request, args *BlockNumberArgs, reply *BlockNumberReply) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError("RPC: Failed to get block number")
		return fmt.Errorf("failed to access the database: %v", err)
	}

	// LevelDB에서 데이터 가져오기
	value, err := dbInstance.Get([]byte("lastblock"), nil)
	if err != nil {
		return fmt.Errorf("failed to get lastblock value: %v", err)
	}

	// JSON 파싱
	var lastBlock blockchain.Block
	err = json.Unmarshal(value, &lastBlock)
	if err != nil {
		return fmt.Errorf("failed to parse block data: %v", err)
	}

	// 블록 번호 가져오기
	reply.BlockNumber = lastBlock.Number
	return nil
}

type LastBlockArgs struct{}
type LastBlockReply struct {
	LastBlock blockchain.Block `json:"lastBlock"`
}

// 가장 최근 블록 조회
func (b *BlockAPI) GetLastBlock(r *http.Request, args *LastBlockArgs, reply *LastBlockReply) error {
	dbInstance, err := leveldb.GetDBInstance()
	if err != nil {
		utils.PrintError("RPC: Failed to get lastblock")
		return fmt.Errorf("failed to access the database: %v", err)
	}

	lastblockData, err := dbInstance.Get([]byte("lastblock"), nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve lastblock value: %v", err)
	}

	var lastBlock blockchain.Block
	err = json.Unmarshal(lastblockData, &lastBlock)
	if err != nil {
		return fmt.Errorf("failed to parse block data: %v", err)
	}

	reply.LastBlock = lastBlock
	return nil
}
