package npool

// Реестр объектов с доступом по строковому ключу
type NamedPool interface {
	Get(key interface{}) interface{}
}

type Sleeper func()

type Wakeupper interface {
	// Вызывается один раз после создания объекта через new()
	New(key interface{}, sleep Sleeper, conf interface{})
	// Вызывается для сброса состояния объекта к исходному, может вызываться много раз
	Wakeup(key interface{}, sleep Sleeper)
}

// Создаёт новый пул
func New(sample Wakeupper, conf interface{}) NamedPool { return newReg(sample, conf) }
