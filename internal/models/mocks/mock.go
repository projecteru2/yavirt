package mocks

import "github.com/projecteru2/yavirt/internal/models"

func Mock() (*Manageable, func()) {
	var origManager = models.GetManager()
	var mockManager = &Manageable{}
	models.SetManager(mockManager)
	return mockManager, func() { models.SetManager(origManager) }
}
