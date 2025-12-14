package option

// ExchangeOptions 交易所配置选项（用于 Exchange 初始化）
type ExchangeOptions struct {
	APIKey    string
	SecretKey string
	Password  string // 密码（用于 OKX 等需要 password 的交易所）
	Sandbox   bool
	Proxy     string
	BaseURL   string
	Debug     bool
	Options   map[string]interface{} // 其他自定义选项
}

// Option 配置选项函数类型（用于 Exchange 初始化）
type Option func(*ExchangeOptions)

// WithAPIKey 设置 API Key
func WithAPIKey(apiKey string) Option {
	return func(opts *ExchangeOptions) {
		opts.APIKey = apiKey
	}
}

// WithSecretKey 设置 Secret Key
func WithSecretKey(secretKey string) Option {
	return func(opts *ExchangeOptions) {
		opts.SecretKey = secretKey
	}
}

// WithPassword 设置 Password（用于 OKX 等需要 password 的交易所）
func WithPassword(password string) Option {
	return func(opts *ExchangeOptions) {
		opts.Password = password
	}
}

// WithSandbox 设置是否使用模拟盘
func WithSandbox(sandbox bool) Option {
	return func(opts *ExchangeOptions) {
		opts.Sandbox = sandbox
	}
}

// WithProxy 设置代理
func WithProxy(proxy string) Option {
	return func(opts *ExchangeOptions) {
		opts.Proxy = proxy
	}
}

// WithBaseURL 设置基础 URL
func WithBaseURL(baseURL string) Option {
	return func(opts *ExchangeOptions) {
		opts.BaseURL = baseURL
	}
}

// WithDebug 设置是否启用调试模式
func WithDebug(debug bool) Option {
	return func(opts *ExchangeOptions) {
		opts.Debug = debug
	}
}

// WithOption 设置自定义选项
func WithOption(key string, value interface{}) Option {
	return func(opts *ExchangeOptions) {
		if opts.Options == nil {
			opts.Options = make(map[string]interface{})
		}
		opts.Options[key] = value
	}
}
