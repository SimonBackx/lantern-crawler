package crawler

// ByAge implements sort.Interface for []Person based on
// the Age field.
type ByIntroduction []*Hostworker

func (a ByIntroduction) Len() int      { return len(a) }
func (a ByIntroduction) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByIntroduction) Less(i, j int) bool {
	return a[i].GetRecrawlDuration() < a[j].GetRecrawlDuration()
}
