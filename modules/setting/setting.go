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
	DbEngine       string `split_words:"true"`
	DbHost         string `split_words:"true"`
	DbPort         int    `split_words:"true"`
	DbUser         string `split_words:"true"`
	DbPassword     string `split_words:"true"`
	DbDatabase     string `split_words:"true"`
	DbPrefix       string `split_words:"true"`
	DbCharset      string `split_words:"true"`
	DbMaxIdleConns int    `split_words:"true"`
	DbMaxOpenConns int    `split_words:"true"`

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
		return ReadIni(cfgKey)
	} else if cfgType == "env" {
		return ReadEnv(cfgKey)
	} else {
		return nil, errors.New("不支持的配置类型")
	}
}

func ReadIni(filename string) (*Setting, error) {
	config, err := ini.Load(filename)
	if err != nil {
		return nil, err
	}
	section := config.Section(DefaultSection)

	var s Setting

	s.DbEngine = section.Key("db.engine").MustString("mysql")
	s.DbHost = section.Key("db.host").MustString("127.0.0.1")
	s.DbPort = section.Key("db.port").MustInt(3306)
	s.DbUser = section.Key("db.user").MustString("")
	s.DbPassword = section.Key("db.password").MustString("")
	s.DbDatabase = section.Key("db.database").MustString("gocron")
	s.DbPrefix = section.Key("db.prefix").MustString("")
	s.DbCharset = section.Key("db.charset").MustString("utf8")
	s.DbMaxIdleConns = section.Key("db.max.idle.conns").MustInt(30)
	s.DbMaxOpenConns = section.Key("db.max.open.conns").MustInt(100)

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

func ReadEnv(prefix string) (*Setting, error) {

	var s = &Setting{}
	err := envconfig.Process(prefix, s)
	if err != nil {
		logger.Debug(err)
	}
	logger.Debug(*s)
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
