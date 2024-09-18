package bankxgo

type Config struct {
	Database struct {
		ConnStr string `yaml:"conn_str"`
	} `yaml:"database"`
	SystemAccounts map[string]string `yaml:"system_accounts"`
	ServiceLimits  ServiceLimitsCfg  `yaml:"service_limits"`
}

type ServiceLimitsCfg struct {
	CreateAccount EndpointLimitCfg `yaml:"create_account"`
	Deposit       EndpointLimitCfg `yaml:"deposit"`
	Withdraw      EndpointLimitCfg `yaml:"withdraw"`
	Balance       EndpointLimitCfg `yaml:"balance"`
	Statement     EndpointLimitCfg `yaml:"statement"`
}

type EndpointLimitCfg struct {
	SloMs int `yaml:"slo_ms"`
	Rate  int `yaml:"rate"`
	Burst int `yaml:"burst"`
}
