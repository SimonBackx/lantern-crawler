package hash

func ToBucket(str []byte, count uint64) int {
	var i uint64
	var q uint64
	q = 1
	for _, b := range str {
		q *= 3
		i += uint64(b) * q
	}
	return int(i % count)
}
