package mb

// Option 可选项
type Option func(client *Client)

// WithReadyQueueSize 就绪队列长度
func WithReadyQueueSize(size int) Option {
	return func(client *Client) {
		client.readyQueueSize = size
	}
}

// WitchHandler 配置handler
func WitchHandler(h Handler) Option {
	return func(client *Client) {
		if h != nil {
			client.handler = h
		}
	}
}

// WitchRetryRandValue 单位ms
// 默认随机值上限,它影响当超时请求入ready队列时,
// 当队列满,会启动一个随机时间rand.Intn(v)*1ms 延迟入队
// 用于需要重试的延迟重试时间
func WitchRetryRandValue(v int) Option {
	return func(client *Client) {
		if v > 0 {
			client.randValue = v
		}
	}
}

// WitchPanicHandle 发生panic回调,主要用于调试
func WitchPanicHandle(f func(interface{})) Option {
	return func(client *Client) {
		if f != nil {
			client.panicHandle = f
		}
	}
}
