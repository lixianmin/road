package road

import (
	"sync"
)

/********************************************************************
created:    2020-09-01
author:     lixianmin

Copyright (C) - All Rights Reserved
*********************************************************************/

type Attachment struct {
	table sync.Map
}

func (my *Attachment) Put(key interface{}, value interface{}) {
	my.table.Store(key, value)
}

func (my *Attachment) UInt32(key interface{}) uint32 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(uint32); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Int32(key interface{}) int32 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(int32); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) UInt64(key interface{}) uint64 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(uint64); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Int64(key interface{}) int64 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(int64); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Int(key interface{}) int {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(int); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Float32(key interface{}) float32 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(float32); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Float64(key interface{}) float64 {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(float64); ok {
			return r
		}
	}

	return 0
}

func (my *Attachment) Bool(key interface{}) bool {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(bool); ok {
			return r
		}
	}

	return false
}

func (my *Attachment) String(key interface{}) string {
	if v, ok := my.Get2(key); ok {
		if r, ok := v.(string); ok {
			return r
		}
	}

	return ""
}

func (my *Attachment) Get1(key interface{}) interface{} {
	if v, ok := my.Get2(key); ok {
		return v
	}

	return nil
}

func (my *Attachment) Get2(key interface{}) (interface{}, bool) {
	return my.table.Load(key)
}

func (my *Attachment) dispose() {
	my.table.Range(func(key, value interface{}) bool {
		my.table.Delete(key)
		return true
	})
}
