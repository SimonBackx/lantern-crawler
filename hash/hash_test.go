package hash

import (
	"fmt"
	"testing"
)

func TestToBucket(test *testing.T) {
	distribution := make([]int, 100)
	b := []byte("a")

	for len(b) <= 16 {
		pos := len(b) - 1
		for pos >= 12 {
			b[pos] += 1

			if b[pos] == '7' {
				b[pos] = 'a'
				pos--
			} else if b[pos] == 'z' {
				b[pos] = '2'
			} else {
				break
			}
		}
		if pos < 12 {
			b = append(b, 'a')
		}

		bucket := ToBucket(b, 100)
		distribution[bucket]++
	}

	for i, bucket := range distribution {
		fmt.Println(i, "\t\t", bucket)
	}
}
