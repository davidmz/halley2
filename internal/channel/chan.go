package channel

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/davidmz/halley2/internal/npool"
	"github.com/davidmz/halley2/internal/ring"
)

type ChanKey struct {
	Site string
	Name string
}

type Message struct {
	ChanName string          `json:"message"`
	Ord      Ord             `json:"ord"`
	Time     time.Time       `json:"-"`
	Body     json.RawMessage `json:"body"`
}

func NewMessage(b []byte) *Message {
	return &Message{Body: b, Ord: NextOrd(), Time: time.Now()}
}

type Receiver interface {
	ReceiveMessage(*Message)
}

type Channel struct {
	lk             sync.Mutex
	key            ChanKey
	lifeTime       time.Duration
	messageRing    *ring.Ring
	sleepFunc      npool.Sleeper
	subscribers    []Receiver
	cleanerStarted bool
}

type ChanConf struct {
	RingSize int
	TTL      time.Duration
}

func (c *Channel) New(key interface{}, sleepFunc npool.Sleeper, conf interface{}) {
	// conf has type *ChanConf
	// key has type ChanKey
	cnf := conf.(*ChanConf)
	c.key = key.(ChanKey)
	c.messageRing = ring.New(cnf.RingSize)
	c.lifeTime = cnf.TTL
	c.sleepFunc = sleepFunc
}

func (c *Channel) Wakeup(key interface{}, sleepFunc npool.Sleeper) {
	c.key = key.(ChanKey)
	c.sleepFunc = sleepFunc
	c.messageRing.Clean()
	c.subscribers = nil
}

func (c *Channel) doSleepIfNeed() {
	if len(c.subscribers) == 0 && c.messageRing.IsEmpty() {
		c.sleepFunc()
		c.sleepFunc = nil
	}
}

func (c *Channel) Subscribe(rcv Receiver, from Ord) {
	c.lk.Lock()
	defer c.lk.Unlock()

	rng := c.messageRing

	if !rng.IsEmpty() {
		if from < 0 {
			// последняя запись
			m := rng.Last().(*Message)
			go rcv.ReceiveMessage(m)
		} else if from > 0 {
			// все записи с Ord > from
			rng.Each(func(_ int, v interface{}) bool {
				m := v.(*Message)
				if m.Ord > from {
					go rcv.ReceiveMessage(m)
				}
				return true
			})
		}
	}

	c.subscribers = append(c.subscribers, rcv)
}

func (c *Channel) Unsubscribe(rcv Receiver) {
	c.lk.Lock()
	defer c.lk.Unlock()
	defer c.doSleepIfNeed()

	foundIdx := -1
	for i, v := range c.subscribers {
		if v == rcv {
			foundIdx = i
			break
		}
	}
	if foundIdx >= 0 {
		c.subscribers[foundIdx] = nil
		c.subscribers = append(c.subscribers[:foundIdx], c.subscribers[foundIdx+1:]...)
	}
}

func (c *Channel) AddMessage(b []byte) {
	c.lk.Lock()
	defer c.lk.Unlock()

	m := &Message{
		Ord:      NextOrd(),
		ChanName: c.key.Name,
		Body:     b,
	}

	c.messageRing.Append(m)

	if !c.cleanerStarted {
		c.cleanerStarted = true
		time.AfterFunc(c.lifeTime, c.cleanOlds)
	}

	for _, s := range c.subscribers {
		go s.ReceiveMessage(m)
	}
}

func (c *Channel) cleanOlds() {
	c.lk.Lock()
	defer c.lk.Unlock()
	defer c.doSleepIfNeed()

	c.cleanerStarted = false

	t := time.Now().Add(-c.lifeTime)

	for !c.messageRing.IsEmpty() {
		if m := c.messageRing.First().(*Message); m.Time.Before(t) {
			c.messageRing.RemoveFirst()
		} else {
			c.cleanerStarted = true
			time.AfterFunc(c.lifeTime, c.cleanOlds)
			break
		}
	}

}
