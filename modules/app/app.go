package app

import (
	"os"

	"io/ioutil"
	"strconv"
	"strings"

	"gocron/modules/logger"
	"gocron/modules/setting"
	"gocron/modules/utils"
)

var (
	AppDir          string           // 应用根目录
	ConfDir         string           // 配置目录
	LogDir          string           // 日志目录
	DataDir         string           // 存放session等
	AppConfigType   string           //配置类型(.ini或者环境变量)
	AppConfigPrefix string           //如果是环境变量需要prefix
	AppConfig       string           // 应用配置文件
	Installed       bool             // 应用是否安装过
	Setting         *setting.Setting // 应用配置
	VersionId       int              // 版本号
	VersionFile     string           // 版本号文件
)

func InitEnv(versionString, cfgType, cfgPrefix string) {
	logger.InitLogger()
	wd, err := os.Getwd()
	if err != nil {
		logger.Fatal(err)
	}
	AppDir = wd
	ConfDir = AppDir + "/conf"
	LogDir = AppDir + "/log"
	DataDir = AppDir + "/data"
	AppConfigType = cfgType
	AppConfig = ConfDir + "/app.ini"
	AppConfigPrefix = cfgPrefix
	VersionFile = ConfDir + "/.version"
	checkDirExists(ConfDir, LogDir, DataDir)
	Installed = IsInstalled()
	VersionId = ToNumberVersion(versionString)
}

// 判断应用是否已安装
func IsInstalled() bool {
	_, err := os.Stat(ConfDir + "/install.lock")
	if os.IsNotExist(err) {
		return false
	}

	return true
}

// 创建安装锁文件
func CreateInstallLock() error {
	_, err := os.Create(ConfDir + "/install.lock")
	if err != nil {
		logger.Error("创建安装锁文件conf/install.lock失败")
	}

	return err
}

// 更新应用版本号文件
func UpdateVersionFile() {
	err := ioutil.WriteFile(VersionFile,
		[]byte(strconv.Itoa(VersionId)),
		0644,
	)

	if err != nil {
		logger.Fatal(err)
	}
}

// 获取应用当前版本号, 从版本号文件中读取
func GetCurrentVersionId() int {
	if !utils.FileExist(VersionFile) {
		return 0
	}

	bytes, err := ioutil.ReadFile(VersionFile)
	if err != nil {
		logger.Fatal(err)
	}

	versionId, err := strconv.Atoi(strings.TrimSpace(string(bytes)))
	if err != nil {
		logger.Fatal(err)
	}

	return versionId
}

// 把字符串版本号a.b.c转换为整数版本号abc
func ToNumberVersion(versionString string) int {
	v := strings.Replace(versionString, ".", "", -1)
	if len(v) < 3 {
		v += "0"
	}

	versionId, err := strconv.Atoi(v)
	if err != nil {
		logger.Fatal(err)
	}

	return versionId
}

// 检测目录是否存在
func checkDirExists(path ...string) {
	for _, value := range path {
		if !utils.FileExist(value) {
			logger.Fatal(value + "目录不存在或无权限访问")
		}
	}
}
