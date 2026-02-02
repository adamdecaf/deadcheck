package provider

import (
	"context"
	"time"

	"github.com/adamdecaf/deadcheck/internal/config"
	"github.com/moov-io/base/log"
)

type MockClient struct {
	logger log.Logger

	Error error
}

var _ Client = (&MockClient{})

func NewMockClient(logger log.Logger) *MockClient {
	return &MockClient{
		logger: logger,
	}
}

func (m *MockClient) Setup(ctx context.Context, check config.Check) error {
	return m.Error
}

func (m *MockClient) CheckIn(ctx context.Context, check config.Check) (time.Time, error) {
	return time.Now().UTC(), m.Error
}
