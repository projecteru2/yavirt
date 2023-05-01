package manager

import "sync"

type serializer struct {
	sync.Map
}

func newSerializer() *serializer {
	ser := &serializer{}

	// TODO
	// reap invalid elements in the sync.Map.
	go func() {
	}()

	return ser
}

func (s *serializer) Serialize(id string, t *task) *taskNotifier {
	t.done = make(chan struct{})

	actual, _ := s.LoadOrStore(id, newExecution(id))
	exec := actual.(*execution) //nolint
	exec.push(t)

	return &taskNotifier{
		done: t.done,
		task: t,
	}
}
