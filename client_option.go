package modbus

import (
	"time"

	"github.com/goburrow/serial"
)

// ClientProviderOption client provider option for user.
type ClientProviderOption func(ClientProvider)

// WithLogProvider set logger provider.
func WithLogProvider(provider LogProvider) ClientProviderOption {
	return func(p ClientProvider) {
		p.setLogProvider(provider)
	}
}

// WithEnableLogger enable log output when you has set logger.
func WithEnableLogger() ClientProviderOption {
	return func(p ClientProvider) {
		p.LogMode(true)
	}
}

// WithSerialConfig set serial config, only valid on serial.
func WithSerialConfig(config serial.Config) ClientProviderOption {
	return func(p ClientProvider) {
		p.setSerialConfig(config)
	}
}

// WithTCPTimeout set tcp Connect & Read timeout, only valid on TCP.
func WithTCPTimeout(t time.Duration) ClientProviderOption {
	return func(p ClientProvider) {
		p.setTCPTimeout(t)
	}
}
