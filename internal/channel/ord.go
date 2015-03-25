package channel

type Ord int64

var ordGen = make(chan Ord)

func NextOrd() Ord {
	return <-ordGen
}

func init() {
	go func() {
		var n Ord
		for {
			n++
			ordGen <- n
		}
	}()
}
