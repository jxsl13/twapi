package econ

import (
	"context"
	"time"
)

type Option func(*Conn)

// WithContext sets the context for the connection
func WithContext(ctx context.Context) Option {
	return func(c *Conn) {
		c.ctx = ctx
	}
}

// WithMaxReconnectDelay sets the maximum delay between reconnects
func WithMaxReconnectDelay(delay time.Duration) Option {
	return func(c *Conn) {
		c.maxReconnectDelay = delay
	}
}

// WithOnConnectCommands sets the commands to be executed on connect
func WithOnConnectCommands(commands ...string) Option {
	return func(c *Conn) {
		c.authCommandList = commands
	}
}
