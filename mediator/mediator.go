package mediator

import "sync"

type Mediator struct {
	RPCToBlockchain chan string // rpc => blockchain
	BlockchainToRPC chan string // blockchain => rpc

	P2PToBlockchain chan string // p2p => blockchain
	BlockchainToP2P chan string // blockchain => p2p
}

var instance *Mediator
var once sync.Once

// GetMediatorInstance
func GetMediatorInstance() *Mediator {
	once.Do(func() {
		instance = &Mediator{
			P2PToBlockchain: make(chan string, 100),
			BlockchainToP2P: make(chan string, 100),
			RPCToBlockchain: make(chan string, 100),
			BlockchainToRPC: make(chan string, 100),
		}
	})
	return instance
}
