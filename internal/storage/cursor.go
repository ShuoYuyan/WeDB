package storage

import (
	"fmt"
)

// Cursor B-Tree 游标
// 用于遍历 B-Tree 中的数据
type Cursor struct {
	tree   *BTree
	stack  []*cursorStackEntry // 游标栈，用于记录路径
	eof    bool                // 是否到达末尾
}

// cursorStackEntry 游标栈条目
type cursorStackEntry struct {
	pageNum  int
	page     *Page
	cellIdx  int // 当前单元格索引
}

// NewCursor 创建新游标
func NewCursor(tree *BTree) *Cursor {
	return &Cursor{
		tree:  tree,
		stack: make([]*cursorStackEntry, 0),
		eof:   false,
	}
}

// First 移动到第一个单元格
// 导航采用“子槽位”模型：内部节点条目的 cellIdx 表示当前所处的
// 子槽 k（k<len(Cells) 时对应 Cells[k].LeftChild，k==len(Cells)
// 对应 Header.RightChild）；叶子节点条目即为单元格下标。
func (c *Cursor) First() error {
	defer func() {
		if r := recover(); r != nil {
			c.Close()
			panic(r)
		}
	}()

	c.stack = c.stack[:0]
	c.eof = false

	page, err := c.tree.GetPage(c.tree.rootPage)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to get root page: %w", err)
	}

	for {
		c.stack = append(c.stack, &cursorStackEntry{
			pageNum: page.PageNum,
			page:    page,
			cellIdx: 0,
		})
		if page.Header.IsLeaf() || len(page.Cells) == 0 {
			break
		}
		page, err = c.tree.GetPage(int(page.Cells[0].LeftChild))
		if err != nil {
			c.Close()
			return fmt.Errorf("failed to get child page: %w", err)
		}
	}

	if e := c.top(); e != nil && e.page.Header.IsLeaf() && len(e.page.Cells) == 0 {
		c.eof = true
	}
	if len(c.stack) == 0 {
		c.eof = true
	}
	return nil
}

// childAt 返回内部页面第 k 个子指针
func childAt(page *Page, k int) (int, bool) {
	if k < 0 {
		return 0, false
	}
	if k < len(page.Cells) {
		lc := int(page.Cells[k].LeftChild)
		return lc, lc > 0
	}
	if k == len(page.Cells) {
		rc := int(page.Header.RightChild)
		return rc, rc > 0
	}
	return 0, false
}

func (c *Cursor) top() *cursorStackEntry {
	if len(c.stack) == 0 {
		return nil
	}
	return c.stack[len(c.stack)-1]
}

// descend 从指定页沿最左路径压栈到叶子
func (c *Cursor) descend(pageNum int) error {
	page, err := c.tree.GetPage(pageNum)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to get child page: %w", err)
	}
	for {
		c.stack = append(c.stack, &cursorStackEntry{
			pageNum: page.PageNum,
			page:    page,
			cellIdx: 0,
		})
		if page.Header.IsLeaf() || len(page.Cells) == 0 {
			return nil
		}
		page, err = c.tree.GetPage(int(page.Cells[0].LeftChild))
		if err != nil {
			c.Close()
			return fmt.Errorf("failed to get child page: %w", err)
		}
	}
}

// Next 移动到下一个单元格
func (c *Cursor) Next() error {
	if c.eof {
		return nil
	}
	cur := c.top()
	if cur == nil {
		c.eof = true
		return nil
	}

	// 叶子内还有下一个单元格
	if cur.page.Header.IsLeaf() && cur.cellIdx < len(cur.page.Cells)-1 {
		cur.cellIdx++
		return nil
	}

	// 爬升：寻找拥有“更靠右子槽”的祖先。
	// 注意：叶子条目的 cellIdx 是单元格下标；作为子槽的应是父条目当前记录的槽位。
	c.stack = c.stack[:len(c.stack)-1]
	for len(c.stack) > 0 {
		e := c.top()
		next := e.cellIdx + 1
		if ptr, ok := childAt(e.page, next); ok {
			e.cellIdx = next
			return c.descend(ptr)
		}
		c.stack = c.stack[:len(c.stack)-1]
	}

	c.eof = true
	return nil
}

// Last 移动到最后一个单元格
func (c *Cursor) Last() error {
	c.stack = c.stack[:0]
	c.eof = false

	page, err := c.tree.GetPage(c.tree.rootPage)
	if err != nil {
		return fmt.Errorf("failed to get root page: %w", err)
	}

	for {
		slot := len(page.Cells) // 默认进入最右子槽
		if page.Header.IsLeaf() || len(page.Cells) == 0 {
			slot = len(page.Cells) - 1 // 叶子：最后一个单元格
			if slot < 0 {
				slot = 0
			}
		}
		c.stack = append(c.stack, &cursorStackEntry{
			pageNum: page.PageNum,
			page:    page,
			cellIdx: slot,
		})
		if page.Header.IsLeaf() {
			break
		}
		ptr, ok := childAt(page, slot)
		if !ok || ptr == 0 {
			break
		}
		page, err = c.tree.GetPage(ptr)
		if err != nil {
			return fmt.Errorf("failed to get child page: %w", err)
		}
	}

	if len(c.stack) == 0 {
		c.eof = true
	}
	return nil
}

// Prev 移动到上一个单元格
func (c *Cursor) Prev() error {
	if c.eof {
		return nil
	}
	cur := c.top()
	if cur == nil {
		c.eof = true
		return nil
	}

	// 叶子内还有上一个单元格
	if cur.page.Header.IsLeaf() && cur.cellIdx > 0 {
		cur.cellIdx--
		return nil
	}

	// 爬升：寻找拥有“更靠左子槽”的祖先
	c.stack = c.stack[:len(c.stack)-1]
	for len(c.stack) > 0 {
		e := c.top()
		prev := e.cellIdx - 1
		if prev >= 0 {
			if ptr, ok := childAt(e.page, prev); ok {
				e.cellIdx = prev
				// 进入该子树的最右叶子
				page, err := c.tree.GetPage(ptr)
				if err != nil {
					c.Close()
					return fmt.Errorf("failed to get child page: %w", err)
				}
				for {
					slot := len(page.Cells) - 1
					if slot < 0 {
						slot = 0
					}
					c.stack = append(c.stack, &cursorStackEntry{
						pageNum: page.PageNum,
						page:    page,
						cellIdx: slot,
					})
					if page.Header.IsLeaf() {
						return nil
					}
					ptr2, ok2 := childAt(page, len(page.Cells))
					if !ok2 || ptr2 == 0 {
						return nil
					}
					page, err = c.tree.GetPage(ptr2)
					if err != nil {
						c.Close()
						return fmt.Errorf("failed to get child page: %w", err)
					}
				}
			}
		}
		c.stack = c.stack[:len(c.stack)-1]
	}

	c.eof = true
	return nil
}
// Seek 移动到指定键
func (c *Cursor) Seek(key int64) error {
	c.stack = c.stack[:0]
	c.eof = false

	// 从根页面开始
	page, err := c.tree.GetPage(c.tree.rootPage)
	if err != nil {
		return fmt.Errorf("failed to get root page: %w", err)
	}

	for {
		// 查找键的位置
		cellIdx := 0
		for i, cell := range page.Cells {
			if key <= cell.Key {
				cellIdx = i
				break
			}
			cellIdx = i + 1
		}

		c.stack = append(c.stack, &cursorStackEntry{
			pageNum: page.PageNum,
			page:    page,
			cellIdx: cellIdx,
		})

		if page.Header.IsLeaf() {
			break
		}

		// 移动到子节点
		var childPageNum int
		if cellIdx < len(page.Cells) {
			childPageNum = int(page.Cells[cellIdx].LeftChild)
		} else {
			childPageNum = int(page.Header.RightChild)
		}

		if childPageNum == 0 {
			break
		}

		page, err = c.tree.GetPage(childPageNum)
		if err != nil {
			return fmt.Errorf("failed to get child page: %w", err)
		}
	}

	if len(c.stack) == 0 {
		c.eof = true
	}

	return nil
}

// EOF 判断是否到达末尾
func (c *Cursor) EOF() bool {
	return c.eof
}

// Key 获取当前键
func (c *Cursor) Key() (int64, error) {
	if c.eof || len(c.stack) == 0 {
		return 0, fmt.Errorf("cursor is at EOF")
	}

	entry := c.stack[len(c.stack)-1]
	page := entry.page

	if entry.cellIdx >= len(page.Cells) {
		return 0, fmt.Errorf("invalid cell index")
	}

	return page.Cells[entry.cellIdx].Key, nil
}

// Data 获取当前数据
func (c *Cursor) Data() ([]byte, error) {
	if c.eof || len(c.stack) == 0 {
		return nil, fmt.Errorf("cursor is at EOF")
	}

	entry := c.stack[len(c.stack)-1]
	page := entry.page

	if entry.cellIdx >= len(page.Cells) {
		return nil, fmt.Errorf("invalid cell index")
	}

	return page.Cells[entry.cellIdx].Data, nil
}

// Cell 获取当前单元格
func (c *Cursor) Cell() (*Cell, error) {
	if c.eof || len(c.stack) == 0 {
		return nil, fmt.Errorf("cursor is at EOF")
	}

	entry := c.stack[len(c.stack)-1]
	page := entry.page

	if entry.cellIdx >= len(page.Cells) {
		return nil, fmt.Errorf("invalid cell index")
	}

	return page.Cells[entry.cellIdx], nil
}

// Delete 删除当前单元格
func (c *Cursor) Delete() error {
	if c.eof || len(c.stack) == 0 {
		return fmt.Errorf("cursor is at EOF")
	}

	entry := c.stack[len(c.stack)-1]
	page := entry.page

	if entry.cellIdx >= len(page.Cells) {
		return fmt.Errorf("invalid cell index")
	}

	// 删除单元格
	if err := page.DeleteCell(page.Cells[entry.cellIdx].Key); err != nil {
		return fmt.Errorf("failed to delete cell: %w", err)
	}

	// 调整索引
	if entry.cellIdx >= len(page.Cells) {
		// 如果删除的是最后一个单元格，尝试移动到前一个
		if entry.cellIdx > 0 {
			entry.cellIdx--
		} else {
			// 如果是第一个单元格且已删除，需要向上查找
			c.stack = c.stack[:len(c.stack)-1]
			if len(c.stack) == 0 {
				c.eof = true
			}
		}
	}

	return nil
}

// Close 关闭游标
func (c *Cursor) Close() {
	c.stack = c.stack[:0]
	c.eof = true
}