package npool_test

import (
	"log"
	"testing"

	"github.com/davidmz/halley2/internal/npool"
	"github.com/stretchr/testify/require"
)

type Obj struct {
	K     string
	S     string
	Sleep npool.Sleeper
}

func (o *Obj) New(key interface{}, sleep npool.Sleeper, conf interface{}) {
	log.Println("NW", key, conf)
	o.K = key.(string)
	o.S = conf.(string)
	o.Sleep = sleep
}

func (o *Obj) Wakeup(key interface{}, sleep npool.Sleeper) {
	log.Println("WK", key)
	o.K = key.(string)
	o.S = "resetted"
	o.Sleep = sleep
}

func getObjFromPool(t *testing.T, pool npool.NamedPool, key string) *Obj {
	x := pool.Get(key)
	require.NotNil(t, x)
	o := x.(*Obj)
	require.IsType(t, (*Obj)(nil), o)
	return o
}

func TestNamedPool(t *testing.T) {
	pool := npool.New((*Obj)(nil), "new")
	o := getObjFromPool(t, pool, "a")
	// только что созданный объект
	require.Equal(t, "new", o.S)
	require.Equal(t, "a", o.K)
	o.Sleep()

	o = getObjFromPool(t, pool, "b")
	// восстановленный объект
	require.Equal(t, "resetted", o.S)
	require.Equal(t, "b", o.K)

	o = getObjFromPool(t, pool, "a")
	require.Equal(t, "new", o.S)
	require.Equal(t, "a", o.K)
}
