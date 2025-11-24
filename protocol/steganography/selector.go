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
	// ВАЖНО: На данный момент реализованы только PlayerMove и CustomPayload
	// ChatMessage, PlayerAction, HandSwing пока не реализованы
	switch {
	case dataSize > MaxDataPerPlayerMove:
		// Данные больше 9 байт - используем CustomPayload (до 32KB)
		return minecraft.PacketTypeCustomPayload

	default:
		// Данные <= 9 байт - используем PlayerMove
		return minecraft.PacketTypePlayerMove
	}
}

// GetMaxPayload возвращает максимальный размер полезной нагрузки для типа пакета
func (ps *PacketSelector) GetMaxPayload(packetType minecraft.PacketType) int {
	switch packetType {
	case minecraft.PacketTypeCustomPayload:
		return 32760 // 32KB - заголовок фрейма

	case minecraft.PacketTypePlayerMove:
		return MaxDataPerPlayerMove // 9 байт

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
