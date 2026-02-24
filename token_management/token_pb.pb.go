package token_management

// This file mirrors the message structures from token_pb.proto.
// It is intentionally lightweight and JSON-tagged for this project runtime.

type Token struct {
	TokenId int32 `json:"token_id"`
}

type WriteTokenMsg struct {
	TokenId      int32  `json:"token_id"`
	PartialValue uint64 `json:"partial_value"`
	Name         string `json:"name"`
	Low          uint64 `json:"low"`
	Mid          uint64 `json:"mid"`
	High         uint64 `json:"high"`
}

type WriteResponse struct {
	Ack      int32  `json:"ack"`
	Message  string `json:"message"`
	Wts      string `json:"wts"`
	FinalVal uint64 `json:"finalVal"`
	Name     string `json:"name"`
	Low      uint64 `json:"low"`
	Mid      uint64 `json:"mid"`
	High     uint64 `json:"high"`
	TokenId  int32  `json:"token_id"`
	Server   string `json:"server"`
}

type WriteBroadcastRequest struct {
	HashVal     uint64 `json:"hash_val"`
	ReadingFlag bool   `json:"reading_flag"`
	Ack         int32  `json:"ack"`
	Wts         string `json:"wts"`
	Name        string `json:"name"`
	Low         uint64 `json:"low"`
	Mid         uint64 `json:"mid"`
	High        uint64 `json:"high"`
	TokenId     int32  `json:"token_id"`
	Server      string `json:"server"`
}

type WriteBroadcastResponse struct {
	Ack int32 `json:"ack"`
}

type ReadBroadcastRequest struct {
	TokenId int32 `json:"token_id"`
}

type ReadBroadcastResponse struct {
	Wts      string `json:"wts"`
	FinalVal uint64 `json:"finalVal"`
	Name     string `json:"name"`
	Low      uint64 `json:"low"`
	Mid      uint64 `json:"mid"`
	High     uint64 `json:"high"`
}
