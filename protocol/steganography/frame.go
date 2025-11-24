package steganography

// Frame представляет мультиплексированный фрейм данных
type Frame struct {
	StreamID uint16 // Идентификатор виртуального потока (0-65535)
	Sequence uint16 // Порядковый номер фрейма
	Flags    uint8  // Управляющие флаги
	Length   uint16 // Длина полезных данных
	Data     []byte // Полезные данные
}

// Флаги фрейма
const (
	FlagSYN uint8 = 1 << 0 // 0x01 - открытие потока
	FlagACK uint8 = 1 << 1 // 0x02 - подтверждение
	FlagFIN uint8 = 1 << 2 // 0x04 - закрытие потока
	FlagRST uint8 = 1 << 3 // 0x08 - сброс потока
	FlagPSH uint8 = 1 << 4 // 0x10 - push data immediately
)

// HeaderSize размер заголовка фрейма
const HeaderSize = 7 // StreamID(2) + Sequence(2) + Flags(1) + Length(2)

// HasFlag проверяет наличие флага
func (f *Frame) HasFlag(flag uint8) bool {
	return f.Flags&flag != 0
}

// SetFlag устанавливает флаг
func (f *Frame) SetFlag(flag uint8) {
	f.Flags |= flag
}

// ClearFlag очищает флаг
func (f *Frame) ClearFlag(flag uint8) {
	f.Flags &= ^flag
}

// Size возвращает полный размер фрейма (заголовок + данные)
func (f *Frame) Size() int {
	return HeaderSize + len(f.Data)
}
