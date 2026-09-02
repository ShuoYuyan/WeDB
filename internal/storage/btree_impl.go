package storage

import (
	"fmt"
	"sync"
)

// BTree B-Tree 实现
// 对应 SQLite 的 Btree 结构
type BTree struct {
	rootPage     int          // 根页面号
	pageSize     int          // 页面大小
	isTable      bool         // 是否为表 B-Tree（整数键）
	pager        *Pager       // Pager 事务管理器
	cache        *PageCache   // 页面缓存
	onRootChange func(int)    // 根页迁移回调（分裂产生新根时调用，持久化映射）
	mu           sync.RWMutex // 读写锁
}

// SetRootChangeCallback 注册根页迁移回调
func (bt *BTree) SetRootChangeCallback(cb func(int)) {
	bt.mu.Lock()
	bt.onRootChange = cb
	bt.mu.Unlock()
}

func (bt *BTree) notifyRootChange(newRoot int) {
	if bt.onRootChange != nil {
		bt.onRootChange(newRoot)
	}
}

// NewBTree 创建新 B-Tree
func NewBTree(rootPage int, pageSize int, isTable bool, pager *Pager, cache *PageCache) *BTree {
	return &BTree{
		rootPage: rootPage,
		pageSize: pageSize,
		isTable:  isTable,
		pager:    pager,
		cache:    cache,
	}
}

// Insert 插入键值对
func (bt *BTree) Insert(key int64, data []byte) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	cell := &Cell{
		Key:  key,
		Data: data,
	}

	// 获取根页面
	root, err := bt.cache.Get(bt.rootPage)
	if err != nil {
		return fmt.Errorf("failed to get root page: %w", err)
	}

	// 如果根页面已满，分裂根页面
	if root.IsFull() {
		// 创建新的根页面
		newRootPageNum := bt.pager.AllocPage()
		newRoot := NewPage(newRootPageNum, bt.pageSize, false)

		// 选择中间单元格作为分隔键
		mid := len(root.Cells) / 2
		separatorKey := root.Cells[mid].Key

		// 将旧根页面作为左子节点
		newRoot.Cells = append(newRoot.Cells, &Cell{
			Key:       separatorKey,
			LeftChild: uint32(bt.rootPage),
		})

		// 分裂旧根页面
		newPage, err := root.Split()
		if err != nil {
			return fmt.Errorf("failed to split root page: %w", err)
		}

		// 分配新页面的页面号
		newPage.PageNum = bt.pager.AllocPage()

		// 更新右子节点
		newRoot.Header.RightChild = uint32(newPage.PageNum)

		// 将新页面添加到缓存
		if err := bt.cache.Put(newRoot); err != nil {
			return fmt.Errorf("failed to put new root page: %w", err)
		}
		if err := bt.cache.Put(newPage); err != nil {
			return fmt.Errorf("failed to put new page: %w", err)
		}

		// 标记页面为脏
		bt.cache.MarkDirty(newRoot)
		bt.cache.MarkDirty(root)
		bt.cache.MarkDirty(newPage)

		// 更新根页面并通知持久化映射（关键：否则后续操作从旧根读取，丢失右子树数据）
		bt.rootPage = newRoot.PageNum
		bt.notifyRootChange(bt.rootPage)
		root = newRoot

		// 重新插入单元格到正确的子节点
		return bt.insert(bt.rootPage, cell)
	}

	// 递归插入
	return bt.insert(bt.rootPage, cell)
}

// BatchInsert 批量插入键值对
func (bt *BTree) BatchInsert(cells []*Cell) error {
	if len(cells) == 0 {
		return nil
	}

	bt.mu.Lock()
	defer bt.mu.Unlock()

	// 批量插入所有单元格
	for _, cell := range cells {
		// 获取根页面
		root, err := bt.cache.Get(bt.rootPage)
		if err != nil {
			return fmt.Errorf("failed to get root page: %w", err)
		}

		// 如果根页面已满，分裂根页面
		if root.IsFull() {
			// 创建新的根页面
			newRootPageNum := bt.pager.AllocPage()
			newRoot := NewPage(newRootPageNum, bt.pageSize, false)

			// 选择中间单元格作为分隔键
			mid := len(root.Cells) / 2
			separatorKey := root.Cells[mid].Key

			// 将旧根页面作为左子节点
			newRoot.Cells = append(newRoot.Cells, &Cell{
				Key:       separatorKey,
				LeftChild: uint32(bt.rootPage),
			})

			// 分裂旧根页面
			newPage, err := root.Split()
			if err != nil {
				return fmt.Errorf("failed to split root page: %w", err)
			}

			// 分配新页面的页面号
			newPage.PageNum = bt.pager.AllocPage()

			// 更新右子节点
			newRoot.Header.RightChild = uint32(newPage.PageNum)

			// 将新页面添加到缓存
			if err := bt.cache.Put(newRoot); err != nil {
				return fmt.Errorf("failed to put new root page: %w", err)
			}
			if err := bt.cache.Put(newPage); err != nil {
				return fmt.Errorf("failed to put new page: %w", err)
			}

			// 标记页面为脏
			bt.cache.MarkDirty(newRoot)
			bt.cache.MarkDirty(root)
			bt.cache.MarkDirty(newPage)

			// 更新根页面
			bt.rootPage = newRoot.PageNum
			bt.notifyRootChange(bt.rootPage)
			root = newRoot
		}

		// 递归插入
		if err := bt.insert(bt.rootPage, cell); err != nil {
			return err
		}
	}

	return nil
}

// insert 递归插入
func (bt *BTree) insert(pageNum int, cell *Cell) error {
	// 获取页面
	page, err := bt.cache.Get(pageNum)
	if err != nil {
		return fmt.Errorf("failed to get page %d: %w", pageNum, err)
	}

	// 如果是叶子页面，直接插入
	if page.Header.IsLeaf() {
		if err := page.InsertCell(cell); err != nil {
			return fmt.Errorf("failed to insert cell: %w", err)
		}
		bt.cache.MarkDirty(page)
		return nil
	}

	// 内部页面，找到合适的子节点（记录槽位：k<len 为 Cells[k].LeftChild，k==len 为 RightChild）
	var childPageNum int
	childSlot := -1
	for k := 0; k <= len(page.Cells); k++ {
		if k < len(page.Cells) {
			if cell.Key <= page.Cells[k].Key {
				childSlot = k
				childPageNum = int(page.Cells[k].LeftChild)
				break
			}
		} else {
			childSlot = k
			childPageNum = int(page.Header.RightChild)
		}
	}

	if childPageNum == 0 {
		kind := "L"
		if !page.Header.IsLeaf() {
			kind = "I"
		}
		return fmt.Errorf("child page not found (page=%d kind=%s cells=%d right=%d key=%d)",
			page.PageNum, kind, len(page.Cells), page.Header.RightChild, cell.Key)
	}

	// 获取子页面
	childPage, err := bt.cache.Get(childPageNum)
	if err != nil {
		return fmt.Errorf("failed to get child page %d: %w", childPageNum, err)
	}

	// 如果子页面已满，分裂
	if childPage.IsFull() {
		// 选择中间单元格作为分隔键
		mid := len(childPage.Cells) / 2
		separatorKey := childPage.Cells[mid].Key

		newPage, err := childPage.Split()
		if err != nil {
			return fmt.Errorf("failed to split page %d: %w", childPageNum, err)
		}

		// 分配新页面的页面号
		newPage.PageNum = bt.pager.AllocPage()

		// 分隔键插入父页面槽位 childSlot：
		//   原: [... , Cells[childSlot].LeftChild=oldChild, ...]
		//   后: [... , {sepKey, oldChild}, newPage 接管原 childSlot+1 的指针 ...]
		separator := &Cell{
			Key:       separatorKey,
			LeftChild: uint32(childPageNum),
		}
		if err := page.InsertCellAt(separator, childSlot); err != nil {
			return fmt.Errorf("failed to insert separator: %w", err)
		}
		// 新右叶继承原 childSlot+1 槽位的指针
		if ptr, ok := childAt(page, childSlot+2); ok {
			newPage.Header.RightChild = uint32(ptr)
		} else {
			newPage.Header.RightChild = 0
		}
		// childSlot+1 槽改指新页
		if childSlot+1 < len(page.Cells) {
			page.Cells[childSlot+1].LeftChild = uint32(newPage.PageNum)
		} else {
			page.Header.RightChild = uint32(newPage.PageNum)
		}

		bt.cache.MarkDirty(page)
		bt.cache.MarkDirty(childPage)
		bt.cache.MarkDirty(newPage)
		if err := bt.cache.Put(newPage); err != nil {
			return fmt.Errorf("failed to cache split page: %w", err)
		}

		// 重新确定子节点
		if cell.Key <= separator.Key {
			childPageNum = int(separator.LeftChild)
		} else {
			childPageNum = int(newPage.PageNum)
		}
	}

	// 递归插入到子节点
	return bt.insert(childPageNum, cell)
}

// Search 搜索键
func (bt *BTree) Search(key int64) ([]byte, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()

	page, err := bt.cache.Get(bt.rootPage)
	if err != nil {
		return nil, fmt.Errorf("failed to get root page: %w", err)
	}

	for {
		// 在当前页面中查找
		cell, found := page.Search(key)
		if found && page.Header.IsLeaf() {
			return cell.Data, nil
		}

		// 如果是叶子页面且未找到
		if page.Header.IsLeaf() {
			return nil, nil
		}

		// 找到子节点
		var childPageNum int
		for i, c := range page.Cells {
			if key <= c.Key {
				childPageNum = int(c.LeftChild)
				break
			}
			if i == len(page.Cells)-1 {
				childPageNum = int(page.Header.RightChild)
			}
		}

		if childPageNum == 0 {
			return nil, nil
		}

		// 移动到子节点
		page, err = bt.cache.Get(childPageNum)
		if err != nil {
			return nil, fmt.Errorf("failed to get child page %d: %w", childPageNum, err)
		}
	}
}

// Delete 删除键
func (bt *BTree) Delete(key int64) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	return bt.delete(bt.rootPage, key)
}

// delete 递归删除
func (bt *BTree) delete(pageNum int, key int64) error {
	// 获取页面
	page, err := bt.cache.Get(pageNum)
	if err != nil {
		return fmt.Errorf("failed to get page %d: %w", pageNum, err)
	}

	// 如果是叶子页面，直接删除
	if page.Header.IsLeaf() {
		if err := page.DeleteCell(key); err != nil {
			return fmt.Errorf("failed to delete cell: %w", err)
		}
		bt.cache.MarkDirty(page)

		// 检查页面是否需要合并
		if len(page.Cells) < bt.minKeys() && pageNum != bt.rootPage {
			if err := bt.balancePage(pageNum); err != nil {
				return fmt.Errorf("failed to balance page: %w", err)
			}
		}

		return nil
	}

	// 内部页面，找到子节点
	var childPageNum int
	for i, c := range page.Cells {
		if key <= c.Key {
			childPageNum = int(c.LeftChild)
			break
		}
		if i == len(page.Cells)-1 {
			childPageNum = int(page.Header.RightChild)
		}
	}

	if childPageNum == 0 {
		return fmt.Errorf("child page not found")
	}

	// 递归删除
	if err := bt.delete(childPageNum, key); err != nil {
		return err
	}

	// 检查子页面是否需要重新平衡
	childPage, err := bt.cache.Get(childPageNum)
	if err != nil {
		return fmt.Errorf("failed to get child page %d: %w", childPageNum, err)
	}

	if len(childPage.Cells) < bt.minKeys() && childPageNum != bt.rootPage {
		if err := bt.balancePage(childPageNum); err != nil {
			return fmt.Errorf("failed to balance child page: %w", err)
		}
	}

	return nil
}

// minKeys 返回最小键数量
func (bt *BTree) minKeys() int {
	return bt.maxKeys()/2
}

// maxKeys 返回最大键数量
func (bt *BTree) maxKeys() int {
	// 计算一个页面能容纳的最大键数
	// 简化计算：假设每个键占用16字节（8字节键 + 8字节数据指针）
	header := NewPageHeader()
	maxCells := (bt.pageSize - header.Size()) / 16
	return maxCells - 1
}

// balancePage 重新平衡页面
func (bt *BTree) balancePage(pageNum int) error {
	// 获取页面的父节点和兄弟节点
	// 简化实现：尝试从右兄弟借一个键
	parent, siblings, err := bt.getSiblings(pageNum)
	if err != nil {
		return err
	}

	if parent == nil || len(siblings) == 0 {
		// 根节点或没有兄弟，直接返回
		return nil
	}

	// 尝试从左兄弟借键
	leftSibling := siblings[0]
	if leftSibling != nil && len(leftSibling.Cells) > bt.minKeys() {
		return bt.borrowFromLeft(pageNum, leftSibling)
	}

	// 尝试从右兄弟借键
	if len(siblings) > 1 && siblings[1] != nil && len(siblings[1].Cells) > bt.minKeys() {
		return bt.borrowFromRight(pageNum, siblings[1])
	}

	// 如果都无法借，则与兄弟合并
	if leftSibling != nil {
		return bt.mergeWithLeft(pageNum, leftSibling)
	}

	return nil
}

// getSiblings 获取页面的父节点和兄弟节点
func (bt *BTree) getSiblings(pageNum int) (*Page, []*Page, error) {
	// 遍历B-Tree找到父节点和兄弟节点
	parent, leftSibling, rightSibling, err := bt.findParentAndSiblings(pageNum, bt.rootPage, nil)
	if err != nil {
		return nil, nil, err
	}

	siblings := make([]*Page, 0, 2)
	if leftSibling != nil {
		siblings = append(siblings, leftSibling)
	}
	if rightSibling != nil {
		siblings = append(siblings, rightSibling)
	}

	return parent, siblings, nil
}

// findParentAndSiblings 递归查找父节点和兄弟节点
func (bt *BTree) findParentAndSiblings(targetPageNum int, currentPageNum int, parent *Page) (*Page, *Page, *Page, error) {
	if currentPageNum == targetPageNum {
		// 找到目标页面，返回父节点
		return parent, nil, nil, nil
	}

	page, err := bt.cache.Get(currentPageNum)
	if err != nil {
		return nil, nil, nil, err
	}

	// 如果是叶子页面，无法继续查找
	if page.Header.IsLeaf() {
		return nil, nil, nil, fmt.Errorf("target page not found")
	}

	// 在内部页面的单元格中查找
	var leftSibling, rightSibling *Page
	
	for i, cell := range page.Cells {
		if targetPageNum == int(cell.LeftChild) {
			// 找到目标页面是当前单元格的左子节点
			// 查找左兄弟
			if i > 0 {
				leftSiblingPageNum := int(page.Cells[i-1].LeftChild)
				leftSibling, err = bt.cache.Get(leftSiblingPageNum)
				if err != nil {
					return nil, nil, nil, err
				}
			}
			
			// 查找右兄弟（当前单元格的右子节点是目标的右兄弟？不，应该是下一个单元格的左子节点）
			if i < len(page.Cells)-1 {
				rightSiblingPageNum := int(page.Cells[i+1].LeftChild)
				rightSibling, err = bt.cache.Get(rightSiblingPageNum)
				if err != nil {
					return nil, nil, nil, err
				}
			} else if page.Header.RightChild != 0 {
				// 目标是最后一个左子节点，右兄弟是RightChild
				rightSiblingPageNum := int(page.Header.RightChild)
				rightSibling, err = bt.cache.Get(rightSiblingPageNum)
				if err != nil {
					return nil, nil, nil, err
				}
			}
			
			return page, leftSibling, rightSibling, nil
		}
		
		// 递归查找子节点
		result, left, right, err := bt.findParentAndSiblings(targetPageNum, int(cell.LeftChild), page)
		if err == nil && result != nil {
			return result, left, right, nil
		}
	}

	// 检查右子节点
	if page.Header.RightChild != 0 {
		result, left, right, err := bt.findParentAndSiblings(targetPageNum, int(page.Header.RightChild), page)
		if err == nil && result != nil {
			return result, left, right, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("target page not found")
}

// borrowFromLeft 从左兄弟借键
func (bt *BTree) borrowFromLeft(pageNum int, leftSibling *Page) error {
	if len(leftSibling.Cells) == 0 {
		return fmt.Errorf("left sibling has no cells to borrow")
	}

	page, err := bt.cache.Get(pageNum)
	if err != nil {
		return err
	}

	// 移动左兄弟的最后一个键到当前页面
	lastKey := leftSibling.Cells[len(leftSibling.Cells)-1]
	
	// 将借来的键插入到当前页面的开头
	page.Cells = append([]*Cell{lastKey}, page.Cells...)

	// 从左兄弟删除
	leftSibling.Cells = leftSibling.Cells[:len(leftSibling.Cells)-1]

	bt.cache.MarkDirty(page)
	bt.cache.MarkDirty(leftSibling)

	return nil
}

// borrowFromRight 从右兄弟借键
func (bt *BTree) borrowFromRight(pageNum int, rightSibling *Page) error {
	// 简化实现：从右兄弟移动第一个键到当前页面
	if len(rightSibling.Cells) == 0 {
		return nil
	}

	page, err := bt.cache.Get(pageNum)
	if err != nil {
		return err
	}

	// 移动右兄弟的第一个键
	firstKey := rightSibling.Cells[0]
	page.Cells = append(page.Cells, firstKey)

	// 从右兄弟删除
	rightSibling.Cells = rightSibling.Cells[1:]

	bt.cache.MarkDirty(page)
	bt.cache.MarkDirty(rightSibling)

	return nil
}

// mergeWithLeft 与左兄弟合并
func (bt *BTree) mergeWithLeft(pageNum int, leftSibling *Page) error {
	page, err := bt.cache.Get(pageNum)
	if err != nil {
		return err
	}

	// 检查合并后是否会溢出
	estimatedSize := 0
	for _, cell := range leftSibling.Cells {
		estimatedSize += 8 + len(cell.Data) + 2
	}
	for _, cell := range page.Cells {
		estimatedSize += 8 + len(cell.Data) + 2
	}

	if estimatedSize > bt.pageSize*7/10 { // 70%阈值
		return fmt.Errorf("merged page would be too large")
	}

	// 合并键
	page.Cells = append(leftSibling.Cells, page.Cells...)

	// 标记左兄弟为已删除（从缓存中移除）
	bt.cache.Remove(leftSibling.PageNum)

	// 释放左兄弟页面号
	bt.pager.FreePage(leftSibling.PageNum)

	// 标记当前页面为脏
	bt.cache.MarkDirty(page)

	return nil
}

// NewCursor 创建新游标
func (bt *BTree) NewCursor() *Cursor {
	return NewCursor(bt)
}

// GetPage 获取页面
func (bt *BTree) GetPage(pageNum int) (*Page, error) {
	return bt.cache.Get(pageNum)
}

// Flush 刷新所有脏页面
func (bt *BTree) Flush() error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	return bt.cache.Flush()
}

// Close 关闭 B-Tree
func (bt *BTree) Close() error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	// 刷新所有脏页面
	if err := bt.cache.Flush(); err != nil {
		return err
	}

	// 清空缓存
	bt.cache.Clear()

	return nil
}