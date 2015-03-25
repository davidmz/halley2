package npool

import (
	"reflect"
	"sync"
)

type reg struct {
	lk    sync.RWMutex
	typ   reflect.Type
	conf  interface{}
	pool  *sync.Pool
	items map[interface{}]Wakeupper
}

func newReg(sample Wakeupper, conf interface{}) *reg {
	r := &reg{
		typ:   reflect.TypeOf(sample).Elem(),
		conf:  conf,
		items: make(map[interface{}]Wakeupper),
		pool:  &sync.Pool{},
	}
	return r
}

func (r *reg) Get(key interface{}) interface{} {
	r.lk.RLock()
	if entry, ok := r.items[key]; ok {
		r.lk.RUnlock()
		return entry
	}

	r.lk.RUnlock()
	return r.createAndGet(key)
}

func (r *reg) createAndGet(key interface{}) interface{} {
	r.lk.Lock()
	defer r.lk.Unlock()

	// Если уже успели создать ключ
	if entry, ok := r.items[key]; ok {
		return entry
	}

	var (
		entry Wakeupper
		sleep = func() { r.delete(key) }
	)
	if x := r.pool.Get(); x == nil {
		entry = reflect.New(r.typ).Interface().(Wakeupper)
		entry.New(key, sleep, r.conf)
	} else {
		entry = x.(Wakeupper)
		entry.Wakeup(key, sleep)
	}

	r.items[key] = entry

	return entry
}

func (r *reg) delete(key interface{}) {
	r.lk.Lock()
	defer r.lk.Unlock()

	if entry, ok := r.items[key]; ok {
		delete(r.items, key)
		r.pool.Put(entry)
	}
}
