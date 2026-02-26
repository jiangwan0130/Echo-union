package errors

import "errors"

// ErrOptimisticLock 乐观锁冲突：记录已被其他操作修改
var ErrOptimisticLock = errors.New("数据已被其他操作修改，请刷新后重试")
