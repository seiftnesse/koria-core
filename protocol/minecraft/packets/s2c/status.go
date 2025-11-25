package s2c

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"koria-core/protocol/minecraft"
)

// StatusResponsePacket - ответ сервера на Status Request (Server List Ping)
// Packet ID: 0x00 (Status state)
type StatusResponsePacket struct {
	JSONResponse string
}

// StatusResponse структура JSON ответа
type StatusResponse struct {
	Version     StatusVersion     `json:"version"`
	Players     StatusPlayers     `json:"players"`
	Description StatusDescription `json:"description"`
	Favicon     string            `json:"favicon,omitempty"`
}

type StatusVersion struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type StatusPlayers struct {
	Max    int                  `json:"max"`
	Online int                  `json:"online"`
	Sample []StatusPlayerSample `json:"sample,omitempty"`
}

type StatusPlayerSample struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type StatusDescription struct {
	Text string `json:"text"`
}

// PacketID возвращает ID пакета
func (p *StatusResponsePacket) PacketID() minecraft.PacketType {
	return 0x00
}

// Encode кодирует пакет
func (p *StatusResponsePacket) Encode(w io.Writer) error {
	// String (VarInt prefixed)
	return minecraft.WriteString(w, p.JSONResponse, 32767)
}

// Decode декодирует пакет
func (p *StatusResponsePacket) Decode(reader io.Reader) error {
	// Для Status Response обычно не нужен decode на сервере
	return fmt.Errorf("decode not implemented for server packet")
}

// PongResponsePacket - ответ на Ping Request
// Packet ID: 0x01 (Status state)
type PongResponsePacket struct {
	Payload int64
}

// PacketID возвращает ID пакета
func (p *PongResponsePacket) PacketID() minecraft.PacketType {
	return 0x01
}

// Encode кодирует пакет
func (p *PongResponsePacket) Encode(w io.Writer) error {
	// Long (8 bytes, big-endian)
	return binary.Write(w, binary.BigEndian, p.Payload)
}

// Decode декодирует пакет
func (p *PongResponsePacket) Decode(reader io.Reader) error {
	// Для Pong Response обычно не нужен decode на сервере
	return fmt.Errorf("decode not implemented for server packet")
}

// NewStatusResponse создает реалистичный Status Response
func NewStatusResponse(serverName string, maxPlayers, onlinePlayers int) *StatusResponsePacket {
	response := StatusResponse{
		Version: StatusVersion{
			Name:     "1.20.4",
			Protocol: 765,
		},
		Players: StatusPlayers{
			Max:    maxPlayers,
			Online: onlinePlayers,
			Sample: []StatusPlayerSample{},
		},
		Description: StatusDescription{
			Text: serverName,
		},
	}

	jsonBytes, _ := json.Marshal(response)

	return &StatusResponsePacket{
		JSONResponse: string(jsonBytes),
	}
}
