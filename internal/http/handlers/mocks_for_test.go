package handlers_test

type fakeMetrics struct {
	c int
}

func (f *fakeMetrics) IncrementActiveClients() { f.c++ }
func (f *fakeMetrics) DecrementActiveClients() { f.c-- }
