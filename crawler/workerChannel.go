package crawler

// The pop channel is a stacked channel used by workers to pop the next URL(s)
// to process.
type WorkerChannel chan []*Hostworker

// Constructor to create and initialize a popChannel
func NewWorkerChannel() WorkerChannel {
	// The pop channel is stacked, so only a buffer of 1 is required
	// see http://gowithconfidence.tumblr.com/post/31426832143/stacked-channels
	return make(chan []*Hostworker, 1)
}

// The stack function ensures the specified URLs are added to the pop channel
// with minimal blocking (since the channel is stacked, it is virtually equivalent
// to an infinitely buffered channel).
// Returns the current length of the stack
func (wc WorkerChannel) stack(worker *Hostworker) int {
	arr := []*Hostworker{worker}

	for {
		select {
		case wc <- arr:
			return len(arr)
		case old := <-wc:
			// Content of the channel got emptied and is now in old, so append whatever
			// is in arr, to it, so that it can either be inserted in the channel,
			// or appended to some other content that got through in the meantime.
			arr = append(old, arr...)
		}
	}
}
