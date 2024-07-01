package dingding

import (
	"context"
	"errors"

	"github.com/blinkbean/dingtalk"
)

type Config struct {
	Token string `toml:"token"`
}

type DingDing struct {
	config Config
	ddtalk *dingtalk.DingTalk
}

func New(cfg Config) (*DingDing, error) {
	if cfg.Token == "" {
		return nil, errors.New("token is required")
	}
	return &DingDing{
		config: cfg,
		ddtalk: dingtalk.InitDingTalk([]string{cfg.Token}, "."),
	}, nil
}

func (dd *DingDing) SendMarkdown(_ context.Context, title, content string) error {
	return dd.ddtalk.SendMarkDownMessage(title, content)
}

func (dd *DingDing) Send(_ context.Context, title, content string) error {
	text := title + "\n" + content
	return dd.ddtalk.SendTextMessage(text)
}
