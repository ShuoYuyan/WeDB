package storage

import (
	"encoding/binary"
	"fmt"
)

// PageHeader B-Tree 页面头
// 对应 SQLite 的 PageHeader 结构
type PageHeader struct {
	// 页面类型
	Flags      uint8
	// 第一个空闲块的偏移
	Free       uint16
	// 页面中单元格的数量
	CellCount  uint16
	// 单元格内容区的起始位置
	CellPtr    uint16
	// 右子节点（对于内部页面）
	RightChild uint32
	// 页面校验和（用于检测页面损坏）
	Checksum   uint32
}

// CellSize 单元格大小（2 字节）
const CellSize = 2

// 页面标志
const (
	PageFlagLeaf      = 0x08 // 叶子页面
	PageFlagIntKey    = 0x01 // 整数键
	PageFlagZeroData  = 0x02 // 零数据长度
	PageFlagLeafData  = 0x04 // 叶子数据
)

// NewPageHeader 创建新的页面头
func NewPageHeader() *PageHeader {
	return &PageHeader{
		Flags:      PageFlagLeaf | PageFlagIntKey,
		Free:       15, // 页面头大小（包含校验和）
		CellCount:  0,
		CellPtr:    0, // 从页面末尾开始
		RightChild: 0,
		Checksum:   0,
	}
}

// IsLeaf 判断是否为叶子页面
func (ph *PageHeader) IsLeaf() bool {
	return ph.Flags&PageFlagLeaf != 0
}

// IsTable 判断是否为表页面（整数键）
func (ph *PageHeader) IsTable() bool {
	return ph.Flags&PageFlagIntKey != 0
}

// Size 返回页面头大小
func (ph *PageHeader) Size() int {
	return 15 // 页面头大小为 15 字节（添加校验和字段）
}

// Serialize 序列化页面头到字节
func (ph *PageHeader) Serialize() []byte {
	buf := make([]byte, 15)
	buf[0] = ph.Flags
	binary.LittleEndian.PutUint16(buf[1:3], ph.Free)
	binary.LittleEndian.PutUint16(buf[3:5], ph.CellCount)
	binary.LittleEndian.PutUint16(buf[5:7], ph.CellPtr)
	binary.LittleEndian.PutUint32(buf[7:11], ph.RightChild)
	binary.LittleEndian.PutUint32(buf[11:15], ph.Checksum)
	return buf
}

// Deserialize 从字节反序列化页面头
func (ph *PageHeader) Deserialize(buf []byte) error {
	if len(buf) < 15 {
		return fmt.Errorf("buffer too small: %d < %d", len(buf), 15)
	}
	ph.Flags = buf[0]
	ph.Free = binary.LittleEndian.Uint16(buf[1:3])
	ph.CellCount = binary.LittleEndian.Uint16(buf[3:5])
	ph.CellPtr = binary.LittleEndian.Uint16(buf[5:7])
	ph.RightChild = binary.LittleEndian.Uint32(buf[7:11])
	ph.Checksum = binary.LittleEndian.Uint32(buf[11:15])
	return nil
}

// Cell B-Tree 单元格
type Cell struct {
	// 键
	Key   int64
	// 数据（叶子页面）
	Data  []byte
	// 左子节点（内部页面）
	LeftChild uint32
}

// SerializeCell 序列化单元格
func SerializeCell(cell *Cell, isLeaf bool) []byte {
	var buf []byte

	// 写入键
	keyBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(keyBytes, uint64(cell.Key))
	buf = append(buf, keyBytes...)

	if isLeaf {
		// 叶子页面：写入数据长度和数据
		dataLen := len(cell.Data)
		lenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBytes, uint32(dataLen))
		buf = append(buf, lenBytes...)
		buf = append(buf, cell.Data...)
	} else {
		// 内部页面：写入左子节点
		childBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(childBytes, cell.LeftChild)
		buf = append(buf, childBytes...)
	}

	return buf
}

// DeserializeCell 反序列化单元格
func DeserializeCell(buf []byte, isLeaf bool) (*Cell, error) {
	cell := &Cell{}

	if len(buf) < 8 {
		return nil, fmt.Errorf("buffer too small for cell")
	}

	// 读取键
	cell.Key = int64(binary.LittleEndian.Uint64(buf[0:8]))
	offset := 8

	if isLeaf {
		// 叶子页面：读取数据长度和数据
		if len(buf) < offset+4 {
			return nil, fmt.Errorf("buffer too small for data length")
		}
		dataLen := binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4

		if len(buf) < offset+int(dataLen) {
			return nil, fmt.Errorf("buffer too small for data")
		}
		cell.Data = buf[offset : offset+int(dataLen)]
	} else {
		// 内部页面：读取左子节点
		if len(buf) < offset+4 {
			return nil, fmt.Errorf("buffer too small for left child")
		}
		cell.LeftChild = binary.LittleEndian.Uint32(buf[offset : offset+4])
	}

	return cell, nil
}

// Page B-Tree 页面
type Page struct {
	PageNum    int
	Data       []byte
	Header     *PageHeader
	Cells      []*Cell
	Dirty      bool
}

// NewPage 创建新页面
func NewPage(pageNum int, pageSize int, isLeaf bool) *Page {
	data := make([]byte, pageSize)

	header := NewPageHeader()
	if isLeaf {
		header.Flags = PageFlagLeaf | PageFlagIntKey
	} else {
		header.Flags = PageFlagIntKey
	}

	// 序列化页面头
	headerBytes := header.Serialize()
	copy(data, headerBytes)

	page := &Page{
		PageNum: pageNum,
		Data:    data,
		Header:  header,
		Cells:   make([]*Cell, 0),
		Dirty:   true,
	}

	return page
}

// DeserializePage 反序列化页面
func DeserializePage(pageNum int, data []byte) (*Page, error) {
	if len(data) < 15 {
		return nil, fmt.Errorf("page too small: %d < %d", len(data), 15)
	}

	page := &Page{
		PageNum: pageNum,
		Data:    make([]byte, len(data)),
		Dirty:   false,
	}
	copy(page.Data, data)

	// 反序列化页面头
	page.Header = NewPageHeader()
	if err := page.Header.Deserialize(data); err != nil {
		return nil, fmt.Errorf("failed to deserialize header: %w", err)
	}

	// 验证页面校验和
	if page.Header.Checksum != 0 {
		computedChecksum := computeChecksum(data[15:])
		if computedChecksum != page.Header.Checksum {
			return nil, fmt.Errorf("page checksum mismatch: expected %x, got %x", page.Header.Checksum, computedChecksum)
		}
	}

	// 验证页面头的合理性
	if page.Header.CellCount > 1000 {
		return nil, fmt.Errorf("invalid cell count: %d", page.Header.CellCount)
	}
	// 空页面的合法状态是 CellPtr == 页大小（单元指针区从页尾向下生长）
	if page.Header.CellPtr > uint16(len(data)) ||
		(page.Header.CellCount > 0 && page.Header.CellPtr >= uint16(len(data))) {
		return nil, fmt.Errorf("invalid cell pointer: %d >= %d", page.Header.CellPtr, len(data))
	}

	// 反序列化单元格
	isLeaf := page.Header.IsLeaf()
	cellPtr := page.Header.CellPtr

	for i := uint16(0); i < page.Header.CellCount; i++ {
		// 读取单元格指针
		if cellPtr+2 > uint16(len(data)) {
			return nil, fmt.Errorf("cell pointer out of bounds at cell %d", i)
		}
		cellOffset := binary.LittleEndian.Uint16(data[cellPtr : cellPtr+2])
		cellPtr += 2

		// 验证单元格偏移量
		if cellOffset >= uint16(len(data)) || cellOffset < uint16(page.Header.Size()) {
			return nil, fmt.Errorf("invalid cell offset at cell %d: %d", i, cellOffset)
		}

		// 读取单元格
		cell, err := DeserializeCell(data[cellOffset:], isLeaf)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize cell %d: %w", i, err)
		}

		page.Cells = append(page.Cells, cell)
	}

	return page, nil
}

// computeChecksum 计算页面数据的校验和
func computeChecksum(data []byte) uint32 {
	// 使用简单的 CRC32 算法
	var crc uint32 = 0xFFFFFFFF
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ 0xEDB88320
			} else {
				crc >>= 1
			}
		}
	}
	return ^crc
}

// Serialize 序列化页面
func (p *Page) Serialize() ([]byte, error) {
	// 创建一个新的缓冲区
	buf := make([]byte, len(p.Data))

	// 计算单元格指针区域（从页面末尾开始）
	cellPtrAreaStart := len(buf) - len(p.Cells)*2
	if cellPtrAreaStart < p.Header.Size() {
		return nil, fmt.Errorf("page overflow: not enough space for cell pointers")
	}

	// 计算单元格数据区域（从页面头之后开始）
	cellDataOffset := p.Header.Size()

	// 按单元格顺序正向写入：指针槽 k 对应 Cells[k]（与 Deserialize 约定一致）
	for i := 0; i < len(p.Cells); i++ {
		cell := p.Cells[i]
		cellBytes := SerializeCell(cell, p.Header.IsLeaf())

		// 检查是否有足够空间
		if cellDataOffset+len(cellBytes) > cellPtrAreaStart {
			return nil, fmt.Errorf("page overflow: not enough space for cell data")
		}

		// 写入单元格数据
		copy(buf[cellDataOffset:cellDataOffset+len(cellBytes)], cellBytes)

		// 写入单元格指针（正序）
		ptrOffset := cellPtrAreaStart + i*2
		binary.LittleEndian.PutUint16(buf[ptrOffset:ptrOffset+2], uint16(cellDataOffset))

		// 更新单元格数据偏移量
		cellDataOffset += len(cellBytes)
	}

	// 更新页面头
	p.Header.CellCount = uint16(len(p.Cells))
	p.Header.CellPtr = uint16(cellPtrAreaStart)
	p.Header.Free = uint16(cellDataOffset - p.Header.Size())

	// 序列化页面头
	headerBytes := p.Header.Serialize()
	copy(buf, headerBytes)

	// 计算并更新校验和（基于页面头之后的所有数据）
	checksum := computeChecksum(buf[p.Header.Size():])
	p.Header.Checksum = checksum

	// 重新序列化页面头以包含校验和
	headerBytes = p.Header.Serialize()
	copy(buf, headerBytes)

	return buf, nil
}

// InsertCell 插入单元格
func (p *Page) InsertCell(cell *Cell) error {
	// 找到插入位置
	pos := 0
	for i, c := range p.Cells {
		if cell.Key <= c.Key {
			pos = i
			break
		}
		pos = i + 1
	}
	return p.InsertCellAt(cell, pos)
}

// InsertCellAt 在指定位置插入单元格（B-Tree 分裂时维护子指针需要精确槽位）
func (p *Page) InsertCellAt(cell *Cell, pos int) error {
	if pos < 0 || pos > len(p.Cells) {
		pos = len(p.Cells)
	}
	p.Cells = append(p.Cells, nil)
	copy(p.Cells[pos+1:], p.Cells[pos:])
	p.Cells[pos] = cell

	p.Header.CellCount = uint16(len(p.Cells))
	p.Dirty = true

	return nil
}

// DeleteCell 删除单元格
func (p *Page) DeleteCell(key int64) error {
	for i, cell := range p.Cells {
		if cell.Key == key {
			p.Cells = append(p.Cells[:i], p.Cells[i+1:]...)
			p.Header.CellCount--
			p.Dirty = true

			// 删除后尝试压缩页面以减少碎片
			p.Compact()

			return nil
		}
	}
	return fmt.Errorf("cell not found")
}

// Compact 压缩页面（整理碎片）
func (p *Page) Compact() {
	if len(p.Cells) == 0 {
		return
	}

	// 重新计算页面大小
	totalSize := p.Header.Size()
	for _, cell := range p.Cells {
		totalSize += 8 + len(cell.Data) + 2 // 键 + 数据 + 指针
	}

	// 如果页面空间利用率低于50%，考虑压缩
	usage := float64(totalSize) / float64(len(p.Data))
	if usage < 0.5 {
		// 页面碎片严重，重新序列化以压缩
		buf, err := p.Serialize()
		if err == nil {
			// 创建新的压缩后的页面
			newPage := &Page{
				PageNum: p.PageNum,
				Data:    buf,
				Cells:   p.Cells,
				Dirty:   true,
			}
			newPage.Header = NewPageHeader()
			newPage.Header.Deserialize(buf)
			p.Data = newPage.Data
			p.Header = newPage.Header
		}
	}
}

// Search 搜索单元格
func (p *Page) Search(key int64) (*Cell, bool) {
	for _, cell := range p.Cells {
		if cell.Key == key {
			return cell, true
		}
	}
	return nil, false
}

// IsFull 检查页面是否已满
func (p *Page) IsFull() bool {
	// 估算页面使用率
	// 如果单元格数据占用超过70%的页面空间，则认为已满
	usedSpace := p.Header.Size() // 页面头占用
	for _, cell := range p.Cells {
		// 估算单元格大小：键(8字节) + 数据(可变) + 指针(2字节)
		cellSize := 8 + len(cell.Data) + 2
		usedSpace += cellSize
	}

	// 计算使用率
	usage := float64(usedSpace) / float64(len(p.Data))
	return usage > 0.7
}

// Split 分裂页面
func (p *Page) Split() (*Page, error) {
	if !p.IsFull() {
		return nil, fmt.Errorf("page is not full")
	}

	// 创建新页面（不设置PageNum，由调用者设置）
	newPage := NewPage(0, len(p.Data), p.Header.IsLeaf())

	// 分裂单元格：中间单元格提升到父页面，不保留在任何一个子页面
	mid := len(p.Cells) / 2

	// 如果是叶子页面，保留mid在左页面
	// 如果是内部页面，mid提升到父页面，不保留在任何子页面
	leftEnd := mid
	if !p.Header.IsLeaf() {
		leftEnd = mid // 内部页面：左页面保留前mid个，mid提升到父页面
	} else {
		leftEnd = mid + 1 // 叶子页面：左页面保留前mid+1个
	}

	// 复制右半部分到新页面
	newPage.Cells = append(newPage.Cells, p.Cells[leftEnd:]...)
	// 保留左半部分在原页面
	p.Cells = p.Cells[:leftEnd]

	// 更新RightChild（对于内部页面）
	if !p.Header.IsLeaf() && len(newPage.Cells) > 0 {
		// 新页面的右子节点需要从原页面继承
		newPage.Header.RightChild = p.Header.RightChild
		// 原页面的右子节点指向新页面（由调用者设置）
	}

	p.Dirty = true
	newPage.Dirty = true

	return newPage, nil
}