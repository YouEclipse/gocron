package setting

import (
	"errors"

	"gocron/modules/logger"
	"gocron/modules/utils"

	"github.com/kelseyhightower/envconfig"

	"gopkg.in/ini.v1"
)

const DefaultSection = "default"

type Setting struct {
	Db struct {
		Engine       string `split_words:"true"`
		Host         string `split_words:"true"`
		Port         int    `split_words:"true"`
		User         string `split_words:"true"`
		Password     string `split_words:"true"`
		Database     string `split_words:"true"`
		Prefix       string `split_words:"true"`
		Charset      string `split_words:"true"`
		MaxIdleConns int    `split_words:"true"`
		MaxOpenConns int    `split_words:"true"`
	}
	AllowIps      string `split_words:"true"`
	AppName       string `split_words:"true"`
	ApiKey        string `split_words:"true"`
	ApiSecret     string `split_words:"true"`
	ApiSignEnable bool   `split_words:"true"`

	EnableTLS bool   `split_words:"true"`
	CAFile    string `split_words:"true"`
	CertFile  string `split_words:"true"`
	KeyFile   string `split_words:"true"`
}

// 读取配置

func Read(cfgKey, cfgType string) (*Setting, error) {
	if cfgType == "ini" {
		return readIni(cfgKey)
	} else if cfgType == "env" {
		return readEnv(cfgKey)
	} else {
		return nil, errors.New("不支持的配置类型")
	}
}

func readIni(filename string) (*Setting, error) {
	config, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}
	section := config.Section(DefaultSection)

	var s Setting

	s.Db.Engine = section.Key("db.engine").MustString("mysql")
	s.Db.Host = section.Key("db.host").MustString("127.0.0.1")
	s.Db.Port = section.Key("db.port").MustInt(3306)
	s.Db.User = section.Key("db.user").MustString("")
	s.Db.Password = section.Key("db.password").MustString("")
	s.Db.Database = section.Key("db.database").MustString("gocron")
	s.Db.Prefix = section.Key("db.prefix").MustString("")
	s.Db.Charset = section.Key("db.charset").MustString("utf8")
	s.Db.MaxIdleConns = section.Key("db.max.idle.conns").MustInt(30)
	s.Db.MaxOpenConns = section.Key("db.max.open.conns").MustInt(100)

	s.AllowIps = section.Key("allow_ips").MustString("")
	s.AppName = section.Key("app.name").MustString("定时任务管理系统")
	s.ApiKey = section.Key("api.key").MustString("")
	s.ApiSecret = section.Key("api.secret").MustString("")
	s.ApiSignEnable = section.Key("api.sign.enable").MustBool(true)

	s.EnableTLS = section.Key("enable_tls").MustBool(false)
	s.CAFile = section.Key("ca_file").MustString("")
	s.CertFile = section.Key("cert_file").MustString("")
	s.KeyFile = section.Key("key_file").MustString("")

	if s.EnableTLS {
		if !utils.FileExist(s.CAFile) {
			logger.Fatalf("failed to read ca cert file: %s", s.CAFile)
		}

		if !utils.FileExist(s.CertFile) {
			logger.Fatalf("failed to read client cert file: %s", s.CertFile)
		}

		if !utils.FileExist(s.KeyFile) {
			logger.Fatalf("failed to read client key file: %s", s.KeyFile)
		}
	}

	return &s, nil
}

func readEnv(prefix string) (*Setting, error) {
	logger.Debug(prefix)
	var s = &Setting{}
	err := envconfig.Process(prefix, s)
	if err != nil {
		logger.Debug(err)
	}
	logger.Debug(s.AppName)
	return s, err
}

// 写入配置
func Write(config []string, filename string) error {
	if len(config) == 0 {
		return errors.New("参数不能为空")
	}
	if len(config)%2 != 0 {
		return errors.New("参数不匹配")
	}

	file := ini.Empty()

	section, err := file.NewSection(DefaultSection)
	if err != nil {
		return err
	}
	for i := 0; i < len(config); {
		_, err = section.NewKey(config[i], config[i+1])
		if err != nil {
			return err
		}
		i += 2
	}
	err = file.SaveTo(filename)

	return err
}
