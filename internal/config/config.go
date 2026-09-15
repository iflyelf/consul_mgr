package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	
	Database struct {
		DSN             string `json:",optional,env=DATABASE_URL"`
		Host            string `json:",optional,default=localhost,env=DB_HOST"`
		Port            int    `json:",optional,default=5432,env=DB_PORT"`
		User            string `json:",optional,default=postgres,env=DB_USER"`
		Password        string `json:",optional,env=DB_PASSWORD"`
		DBName          string `json:",optional,default=consul_mgr,env=DB_NAME"`
		SSLMode         string `json:",optional,default=disable,env=DB_SSLMODE"`
		MaxOpenConns    int    `json:",optional,default=100"`
		MaxIdleConns    int    `json:",optional,default=10"`
		ConnMaxLifetime int    `json:",optional,default=3600"` // seconds
	}
	
	JWT struct {
		Secret        string `json:",env=JWT_SECRET"`
		AccessExpire  int64  `json:",default=7200"`   // 2 hours
		RefreshExpire int64  `json:",default=604800"` // 7 days
		Issuer        string `json:",default=consul_mgr"`
	}
	
	Admin struct {
		Username string `json:",default=admin,env=ADMIN_USERNAME"`
		Password string `json:",env=ADMIN_PASSWORD"`
		Email    string `json:",env=ADMIN_EMAIL"`
	}
	
	Consul struct {
		DefaultAddress    string `json:",optional,env=CONSUL_ADDRESS"`
		DefaultToken      string `json:",optional,env=CONSUL_TOKEN"`
		DefaultDatacenter string `json:",default=dc1,env=CONSUL_DATACENTER"`
		Timeout           int    `json:",default=10"` // seconds
	}
	
	Web struct {
		Embedded  bool   `json:",default=true"`
		StaticDir string `json:",default=./web/dist"`
		Port      int    `json:",default=5173,env=WEB_PORT"`
		Host      string `json:",default=0.0.0.0,env=WEB_HOST"`
	}
	
	LogConfig struct {
		Level  string `json:",default=info,env=LOG_LEVEL"`
		Format string `json:",default=json,env=LOG_FORMAT"`
	}
	
	Audit struct {
		Enabled       bool `json:",default=true,env=AUDIT_ENABLED"`
		RetentionDays int  `json:",default=90,env=AUDIT_RETENTION_DAYS"`
	}
	
	Casdoor struct {
		Endpoint         string `json:",env=CASDOOR_ENDPOINT"`
		PublicEndpoint   string `json:",optional,env=CASDOOR_PUBLIC_ENDPOINT"`
		ClientId         string `json:",env=CASDOOR_CLIENT_ID"`
		ClientSecret     string `json:",env=CASDOOR_CLIENT_SECRET"`
		Certificate      string `json:",optional,env=CASDOOR_CERTIFICATE"`
		OrganizationName string `json:",default=consul_mgr,env=CASDOOR_ORGANIZATION"`
		ApplicationName  string `json:",default=consul_manager,env=CASDOOR_APPLICATION"`
	}
	
	Permission struct {
		EnableServiceGroupAuth bool     `json:",default=true"`
		DefaultPermissions     []string `json:",default=[read]"`
	}
}
