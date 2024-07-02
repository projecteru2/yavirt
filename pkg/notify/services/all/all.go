package all

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/projecteru2/yavirt/pkg/notify"
)

type Config struct {
	Types []string `toml:"types"`
}

type Manager struct {
	notifiers []notify.Service
}

func New(notifiers []notify.Service) *Manager {
	return &Manager{
		notifiers: notifiers,
	}
}

func (m *Manager) Send(ctx context.Context, title, msg string) error {
	var combinedErr error
	for _, n := range m.notifiers {
		if err := n.Send(ctx, title, msg); err != nil {
			combinedErr = errors.CombineErrors(combinedErr, err)
		}
	}
	return combinedErr
}

func (m *Manager) SendMarkdown(ctx context.Context, title, msg string) error {
	var combinedErr error
	for _, n := range m.notifiers {
		if err := n.SendMarkdown(ctx, title, msg); err != nil {
			combinedErr = errors.CombineErrors(combinedErr, err)
		}
	}
	return combinedErr
}
