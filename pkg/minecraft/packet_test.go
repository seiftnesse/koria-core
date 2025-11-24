package minecraft

import (
	"bytes"
	"fmt"
	"testing"
)

func TestVarIntEncoding(t *testing.T) {
	tests := []struct {
		name     string
		value    int32
		expected []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"127", 127, []byte{0x7F}},
		{"128", 128, []byte{0x80, 0x01}},
		{"300", 300, []byte{0xAC, 0x02}},
		{"2097151", 2097151, []byte{0xFF, 0xFF, 0x7F}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := writeVarInt(buf, tt.value)
			if err != nil {
				t.Fatalf("writeVarInt failed: %v", err)
			}

			result := buf.Bytes()
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("writeVarInt(%d) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestVarIntDecoding(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int32
	}{
		{"zero", []byte{0x00}, 0},
		{"one", []byte{0x01}, 1},
		{"127", []byte{0x7F}, 127},
		{"128", []byte{0x80, 0x01}, 128},
		{"300", []byte{0xAC, 0x02}, 300},
		{"2097151", []byte{0xFF, 0xFF, 0x7F}, 2097151},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewReader(tt.input)
			result, err := readVarInt(buf)
			if err != nil {
				t.Fatalf("readVarInt failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("readVarInt(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestVarIntRoundTrip(t *testing.T) {
	values := []int32{0, 1, 127, 128, 255, 256, 1000, 32767, 65535, 2097151}

	for _, val := range values {
		t.Run(fmt.Sprintf("value_%d", val), func(t *testing.T) {
			// Encode
			buf := new(bytes.Buffer)
			err := writeVarInt(buf, val)
			if err != nil {
				t.Fatalf("writeVarInt failed: %v", err)
			}

			// Decode
			result, err := readVarInt(buf)
			if err != nil {
				t.Fatalf("readVarInt failed: %v", err)
			}

			if result != val {
				t.Errorf("Round trip failed: got %d, want %d", result, val)
			}
		})
	}
}

func TestPacketEncoding(t *testing.T) {
	testData := []byte("Hello, Minecraft!")

	encoded, err := EncodePacket(PacketCustomPayload, testData)
	if err != nil {
		t.Fatalf("EncodePacket failed: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded packet is empty")
	}

	// Should contain at least the length prefix, packet ID, and data
	if len(encoded) < len(testData)+2 {
		t.Errorf("Encoded packet too short: got %d bytes, want at least %d", len(encoded), len(testData)+2)
	}
}

func TestPacketDecoding(t *testing.T) {
	testData := []byte("Hello, Minecraft!")
	packetID := PacketCustomPayload

	// Encode first
	encoded, err := EncodePacket(packetID, testData)
	if err != nil {
		t.Fatalf("EncodePacket failed: %v", err)
	}

	// Decode
	buf := bytes.NewReader(encoded)
	packet, err := DecodePacket(buf)
	if err != nil {
		t.Fatalf("DecodePacket failed: %v", err)
	}

	// Verify packet ID
	if packet.ID != packetID {
		t.Errorf("Packet ID mismatch: got %d, want %d", packet.ID, packetID)
	}

	// Verify data
	if !bytes.Equal(packet.Data, testData) {
		t.Errorf("Packet data mismatch: got %v, want %v", packet.Data, testData)
	}
}

func TestPacketRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		packetID PacketID
		data     []byte
	}{
		{"empty", PacketCustomPayload, []byte{}},
		{"small", PacketCustomPayload, []byte("test")},
		{"medium", PacketKeepAlive, []byte("This is a medium sized packet with some data")},
		{"large", PacketCustomPayload, make([]byte, 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded, err := EncodePacket(tt.packetID, tt.data)
			if err != nil {
				t.Fatalf("EncodePacket failed: %v", err)
			}

			// Decode
			buf := bytes.NewReader(encoded)
			packet, err := DecodePacket(buf)
			if err != nil {
				t.Fatalf("DecodePacket failed: %v", err)
			}

			// Verify
			if packet.ID != tt.packetID {
				t.Errorf("Packet ID mismatch: got %d, want %d", packet.ID, tt.packetID)
			}

			if !bytes.Equal(packet.Data, tt.data) {
				t.Errorf("Packet data mismatch")
			}
		})
	}
}

func TestStringEncoding(t *testing.T) {
	tests := []string{
		"",
		"a",
		"Hello",
		"Hello, World!",
		"Unicode: 你好",
	}

	for _, str := range tests {
		t.Run(str, func(t *testing.T) {
			buf := new(bytes.Buffer)

			// Write
			err := WriteString(buf, str)
			if err != nil {
				t.Fatalf("WriteString failed: %v", err)
			}

			// Read
			result, err := ReadString(buf)
			if err != nil {
				t.Fatalf("ReadString failed: %v", err)
			}

			if result != str {
				t.Errorf("String mismatch: got %q, want %q", result, str)
			}
		})
	}
}

func TestCreateHandshakePacket(t *testing.T) {
	packet, err := CreateHandshakePacket("localhost", 25565)
	if err != nil {
		t.Fatalf("CreateHandshakePacket failed: %v", err)
	}

	if len(packet) == 0 {
		t.Error("Handshake packet is empty")
	}

	// Decode to verify it's a valid packet
	buf := bytes.NewReader(packet)
	decoded, err := DecodePacket(buf)
	if err != nil {
		t.Fatalf("Failed to decode handshake packet: %v", err)
	}

	if decoded.ID != PacketHandshake {
		t.Errorf("Expected handshake packet ID, got %d", decoded.ID)
	}
}

func TestCreateKeepAlivePacket(t *testing.T) {
	keepAliveID := int64(12345678)
	packet, err := CreateKeepAlivePacket(keepAliveID)
	if err != nil {
		t.Fatalf("CreateKeepAlivePacket failed: %v", err)
	}

	if len(packet) == 0 {
		t.Error("Keep-alive packet is empty")
	}

	// Decode to verify it's a valid packet
	buf := bytes.NewReader(packet)
	decoded, err := DecodePacket(buf)
	if err != nil {
		t.Fatalf("Failed to decode keep-alive packet: %v", err)
	}

	if decoded.ID != PacketKeepAlive {
		t.Errorf("Expected keep-alive packet ID, got %d", decoded.ID)
	}

	if len(decoded.Data) != 8 {
		t.Errorf("Expected 8 bytes of data for int64, got %d", len(decoded.Data))
	}
}

func BenchmarkPacketEncode(b *testing.B) {
	data := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		_, err := EncodePacket(PacketCustomPayload, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPacketDecode(b *testing.B) {
	data := make([]byte, 1024)
	encoded, _ := EncodePacket(PacketCustomPayload, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewReader(encoded)
		_, err := DecodePacket(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
