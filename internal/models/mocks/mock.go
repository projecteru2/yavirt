package mocks

import "github.com/projecteru2/yavirt/internal/models"

func Mock() (*Manageable, func()) {
	var origManager = model.GetManager()
	var mockManager = &Manageable{}
	model.SetManager(mockManager)
	return mockManager, func() { model.SetManager(origManager) }
}
