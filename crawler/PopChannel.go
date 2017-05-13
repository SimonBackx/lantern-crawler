package crawler

import (
	"net/url"
)

// The pop channel is a stacked channel used by workers to pop the next URL(s)
// to process.
type popChannel chan []*url.URL

// Constructor to create and initialize a popChannel
func newPopChannel() popChannel {
	// The pop channel is stacked, so only a buffer of 1 is required
	// see http://gowithconfidence.tumblr.com/post/31426832143/stacked-channels
	return make(chan []*url.URL, 1)
}

// The stack function ensures the specified URLs are added to the pop channel
// with minimal blocking (since the channel is stacked, it is virtually equivalent
// to an infinitely buffered channel).
// Returns the current length of the stack
func (pc popChannel) stack(item *url.URL) int {
	arr := []*url.URL{item}

	for {
		select {
		case pc <- arr:
			return len(arr)
		case old := <-pc:
			// Content of the channel got emptied and is now in old, so append whatever
			// is in arr, to it, so that it can either be inserted in the channel,
			// or appended to some other content that got through in the meantime.
			arr = append(old, arr...)
		}
	}
}
