package mocks

import "github.com/projecteru2/yavirt/store"

// Mock .
func Mock() (*Store, func()) {
	var ori = store.GetStore()
	var ms = &Store{}
	store.SetStore(ms)
	return ms, func() { store.SetStore(ori) }
}
