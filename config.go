package bankxgo

type Config struct {
	Database struct {
		ConnectionString string            `yaml:"conn_str"`
		SystemAccounts   map[string]string `yaml:"system_accounts"`
	} `yaml:"database"`
}
