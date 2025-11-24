package steganography

import (
	"encoding/binary"
	"fmt"
	c2s "koria-core/protocol/minecraft/packets/c2s"
	"math"
	"math/rand"
)

// Encoder кодирует фреймы в Minecraft пакеты
type Encoder struct {
	rand *rand.Rand
}

// NewEncoder создает новый энкодер
func NewEncoder() *Encoder {
	return &Encoder{
		rand: rand.New(rand.NewSource(rand.Int63())),
	}
}

// EncodeFrame кодирует фрейм в PlayerMovePacket
// Использует стеганографию - прячет данные в младших битах координат
func (e *Encoder) EncodeFrame(frame *Frame) (*c2s.PlayerMovePacket, error) {
	if len(frame.Data) > MaxDataPerPlayerMove {
		return nil, fmt.Errorf("frame data too large: %d > %d", len(frame.Data), MaxDataPerPlayerMove)
	}

	// Генерируем реалистичные базовые координаты
	baseX := e.generateRealisticCoord()
	baseY := e.generateRealisticY()
	baseZ := e.generateRealisticCoord()
	baseYaw := e.rand.Float32() * 360.0
	basePitch := e.rand.Float32()*180.0 - 90.0

	// Подготавливаем данные для кодирования (заголовок + данные)
	encodedData := make([]byte, HeaderSize+len(frame.Data))

	// Заголовок фрейма
	binary.BigEndian.PutUint16(encodedData[0:2], frame.StreamID)
	binary.BigEndian.PutUint16(encodedData[2:4], frame.Sequence)
	encodedData[4] = frame.Flags
	binary.BigEndian.PutUint16(encodedData[5:7], uint16(len(frame.Data)))

	// Данные
	copy(encodedData[7:], frame.Data)

	// Создаем пакет
	pkt := &c2s.PlayerMovePacket{}

	// Кодируем данные в координаты (используем младшие 32 бита мантиссы double)
	// X: первые 4 байта
	if len(encodedData) >= 4 {
		pkt.X = e.encodeDataInDouble(baseX, encodedData[0:4])
	} else {
		pkt.X = baseX
	}

	// Y: следующие 4 байта
	if len(encodedData) >= 8 {
		pkt.Y = e.encodeDataInDouble(baseY, encodedData[4:8])
	} else if len(encodedData) > 4 {
		padded := make([]byte, 4)
		copy(padded, encodedData[4:])
		pkt.Y = e.encodeDataInDouble(baseY, padded)
	} else {
		pkt.Y = baseY
	}

	// Z: следующие 4 байта
	if len(encodedData) >= 12 {
		pkt.Z = e.encodeDataInDouble(baseZ, encodedData[8:12])
	} else if len(encodedData) > 8 {
		padded := make([]byte, 4)
		copy(padded, encodedData[8:])
		pkt.Z = e.encodeDataInDouble(baseZ, padded)
	} else {
		pkt.Z = baseZ
	}

	// Yaw: следующие 2 байта (используем младшие 16 бит float)
	if len(encodedData) >= 14 {
		pkt.Yaw = e.encodeDataInFloat(baseYaw, encodedData[12:14])
	} else if len(encodedData) > 12 {
		padded := make([]byte, 2)
		copy(padded, encodedData[12:])
		pkt.Yaw = e.encodeDataInFloat(baseYaw, padded)
	} else {
		pkt.Yaw = baseYaw
	}

	// Pitch: следующие 2 байта
	if len(encodedData) >= 16 {
		pkt.Pitch = e.encodeDataInFloat(basePitch, encodedData[14:16])
	} else if len(encodedData) > 14 {
		padded := make([]byte, 2)
		copy(padded, encodedData[14:])
		pkt.Pitch = e.encodeDataInFloat(basePitch, padded)
	} else {
		pkt.Pitch = basePitch
	}

	// Flags: последний байт (младшие 2 бита для Minecraft, остальное - наши данные)
	if len(encodedData) >= 17 {
		pkt.Flags = encodedData[16] & 0x03 // Только младшие 2 бита для MC флагов
	} else {
		pkt.Flags = uint8(e.rand.Intn(4)) // Случайные MC флаги
	}

	return pkt, nil
}

const (
	// MaxDataPerPlayerMove максимум данных в PlayerMovePacket после заголовка
	MaxDataPerPlayerMove = 10 // 17 байт всего - 7 байт заголовок = 10 байт данных
)

// encodeDataInDouble кодирует данные в младшие 32 бита мантиссы double
func (e *Encoder) encodeDataInDouble(baseValue float64, data []byte) float64 {
	// Получаем биты double
	bits := math.Float64bits(baseValue)

	// Очищаем младшие 32 бита
	bits = bits & 0xFFFFFFFF00000000

	// Кодируем наши данные в младшие 32 бита
	var dataValue uint32
	if len(data) >= 4 {
		dataValue = binary.BigEndian.Uint32(data)
	} else {
		// Дополняем нулями если данных меньше 4 байт
		padded := make([]byte, 4)
		copy(padded, data)
		dataValue = binary.BigEndian.Uint32(padded)
	}

	bits = bits | uint64(dataValue)

	return math.Float64frombits(bits)
}

// encodeDataInFloat кодирует данные в младшие 16 бит float
func (e *Encoder) encodeDataInFloat(baseValue float32, data []byte) float32 {
	bits := math.Float32bits(baseValue)

	// Очищаем младшие 16 бит
	bits = bits & 0xFFFF0000

	// Кодируем данные
	var dataValue uint16
	if len(data) >= 2 {
		dataValue = binary.BigEndian.Uint16(data)
	} else if len(data) == 1 {
		dataValue = uint16(data[0])
	}

	bits = bits | uint32(dataValue)

	return math.Float32frombits(bits)
}

// generateRealisticCoord генерирует реалистичную координату в пределах Minecraft world border
func (e *Encoder) generateRealisticCoord() float64 {
	// Minecraft world border: ±29,999,984 blocks
	// Но чаще игроки находятся ближе к спавну, сгенерируем в пределах ±10,000
	return (e.rand.Float64()*20000.0 - 10000.0) + e.rand.Float64()*10.0
}

// generateRealisticY генерирует реалистичную Y координату
func (e *Encoder) generateRealisticY() float64 {
	// Y координата от -64 до 320 в современном Minecraft
	// Чаще всего игроки на высоте 60-80
	return 60.0 + e.rand.Float64()*20.0 + e.rand.Float64()*5.0
}

// EncodeFrameInCustomPayload кодирует фрейм в CustomPayloadPacket
// Для больших блоков данных - просто записываем напрямую
func (e *Encoder) EncodeFrameInCustomPayload(frame *Frame) (*c2s.CustomPayloadPacket, error) {
	// Подготавливаем данные (заголовок + данные)
	payload := make([]byte, HeaderSize+len(frame.Data))

	binary.BigEndian.PutUint16(payload[0:2], frame.StreamID)
	binary.BigEndian.PutUint16(payload[2:4], frame.Sequence)
	payload[4] = frame.Flags
	binary.BigEndian.PutUint16(payload[5:7], uint16(len(frame.Data)))
	copy(payload[7:], frame.Data)

	return &c2s.CustomPayloadPacket{
		Channel: "minecraft:brand", // Легитимный канал
		Data:    payload,
	}, nil
}
