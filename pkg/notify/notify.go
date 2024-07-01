package notify

import (
	"context"
)

type Service interface {
	Send(ctx context.Context, title, msg string) error
	SendMarkdown(ctx context.Context, title, msg string) error
}
