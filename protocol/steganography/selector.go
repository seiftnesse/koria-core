package steganography

import (
	"koria-core/protocol/minecraft"
)

// PacketSelector выбирает оптимальный тип пакета для передачи данных
type PacketSelector struct{}

// NewPacketSelector создает новый selector
func NewPacketSelector() *PacketSelector {
	return &PacketSelector{}
}

// SelectPacketType выбирает тип пакета на основе размера данных
func (ps *PacketSelector) SelectPacketType(dataSize int) minecraft.PacketType {
	switch {
	case dataSize > 512:
		// Большие блоки данных - CustomPayload (до 32KB)
		return minecraft.PacketTypeCustomPayload

	case dataSize > 100:
		// Средние блоки - ChatMessage (~256 байт)
		return minecraft.PacketTypeChatMessage

	case dataSize > 10:
		// Мелкие данные - PlayerMove (17 байт полезной нагрузки)
		// Но так как у нас overhead от заголовка (7 байт), реально ~10 байт
		return minecraft.PacketTypePlayerMove

	case dataSize > 4:
		// Очень мелкие данные - PlayerAction (~12 байт)
		return minecraft.PacketTypePlayerAction

	default:
		// Микроданные - HandSwing (1 байт)
		return minecraft.PacketTypeHandSwing
	}
}

// GetMaxPayload возвращает максимальный размер полезной нагрузки для типа пакета
func (ps *PacketSelector) GetMaxPayload(packetType minecraft.PacketType) int {
	switch packetType {
	case minecraft.PacketTypeCustomPayload:
		return 32760 // 32KB - заголовок фрейма

	case minecraft.PacketTypeChatMessage:
		return 200 // ~200 байт после заголовка и служебных полей

	case minecraft.PacketTypePlayerMove:
		return MaxDataPerPlayerMove // 10 байт

	case minecraft.PacketTypePlayerAction:
		return 5 // ~5 байт полезных данных

	case minecraft.PacketTypeHandSwing:
		return 1 // 1 байт

	default:
		return MaxDataPerPlayerMove
	}
}

// ShouldFragmentData проверяет, нужно ли фрагментировать данные
func (ps *PacketSelector) ShouldFragmentData(dataSize int, packetType minecraft.PacketType) bool {
	maxPayload := ps.GetMaxPayload(packetType)
	return dataSize > maxPayload
}

// CalculateFragments вычисляет количество фрагментов для данных
func (ps *PacketSelector) CalculateFragments(dataSize int, packetType minecraft.PacketType) int {
	maxPayload := ps.GetMaxPayload(packetType)
	fragments := dataSize / maxPayload
	if dataSize%maxPayload != 0 {
		fragments++
	}
	return fragments
}
