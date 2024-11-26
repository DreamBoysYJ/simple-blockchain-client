package protocol_constants

const (
	// Node Discovery 상수
	NodeDiscoveryPing        = 0x01
	NodeDiscoveryPong        = 0x02
	NodeDiscoveryFindNode    = 0x03
	NodeDiscoveryNeighbors   = 0x04
	NodeDiscoveryENRRequest  = 0x05
	NodeDiscoveryENRResponse = 0x06

	// RLPx 상수
	RLPx = 0x07

	// P2P Protocol
	P2PTransactionMessage = 0x01
	P2PBlockMessage       = 0x02
)
