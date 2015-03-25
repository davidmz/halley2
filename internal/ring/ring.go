package ring

type Ring struct {
	items []interface{}
	size,
	head,
	length int
}

func New(size int) *Ring {
	return &Ring{
		items: make([]interface{}, size),
		size:  size,
	}
}

// Длина списка
func (r *Ring) Length() int {
	return r.length
}

// Размер списка
func (r *Ring) Size() int {
	return r.size
}

// Пуст ли список?
func (r *Ring) IsEmpty() bool {
	return r.length == 0
}

// n-й элемент списка
func (r *Ring) Item(n int) interface{} {
	if n < 0 || n >= r.length {
		return nil
	}
	return r.items[r.rAdd(r.head, n)]
}

// Добавить элемент в хвост
func (r *Ring) Append(v interface{}) {
	if r.length == r.size { // буфер полон
		r.items[r.head] = v
		r.head = r.rAdd(r.head, 1)
	} else { // буфер не полон
		p := r.rAdd(r.head, r.length)
		r.items[p] = v
		r.length++
	}
}

// Добавить элемент в голову
func (r *Ring) Prepend(v interface{}) {
	r.head = r.rAdd(r.head, -1)
	r.items[r.head] = v
	if r.length < r.size {
		r.length++
	}
}

// Первый элемент
func (r *Ring) First() interface{} {
	return r.Item(0)
}

// Последний элемент
func (r *Ring) Last() interface{} {
	return r.Item(r.length - 1)
}

// Удалить первый элемент
func (r *Ring) RemoveFirst() {
	if r.IsEmpty() {
		return
	}
	r.items[r.head] = nil
	r.head = r.rAdd(r.head, 1)
	r.length--
}

// Удалить последний элемент
func (r *Ring) RemoveLast() {
	if r.IsEmpty() {
		return
	}
	r.items[r.rAdd(r.head, r.length-1)] = nil
	r.length--
}

// Пройтись по всем элементам
func (r *Ring) Each(foo func(int, interface{}) bool) {
	for i := 0; i < r.length; i++ {
		if !foo(i, r.Item(i)) {
			break
		}
	}
}

// Очистить массив
func (r *Ring) Clean() {
	r.head = 0
	r.length = 0
	for i := range r.items {
		r.items[i] = nil
	}
}

//////////////////////////////

func (r *Ring) rAdd(val, shift int) (res int) {
	res = val + shift
	for res >= r.size {
		res -= r.size
	}
	for res < 0 {
		res += r.size
	}
	return
}
