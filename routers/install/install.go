package install

import (
	"fmt"
	"gocron/modules/logger"
	"strconv"

	"gocron/models"
	"gocron/modules/app"
	"gocron/modules/setting"
	"gocron/modules/utils"
	"gocron/service"

	"github.com/go-macaron/binding"
	"gopkg.in/macaron.v1"
)

// 系统安装

type InstallForm struct {
	DbType        string `binding:"In(mysql)"`
	DbHost        string
	DbPort        int
	DbUsername    string
	DbPassword    string
	DbName        string `binding:"MaxSize(50)"`
	DbTablePrefix string `binding:"MaxSize(20)"`
	//ConfigType           string `binding:"In(ini,env)"`
	AdminUsername        string `binding:"Required;MinSize(3)"`
	AdminPassword        string `binding:"Required;MinSize(6)"`
	ConfirmAdminPassword string `binding:"Required;MinSize(6)"`
	AdminEmail           string `binding:"Required;Email;MaxSize(50)"`
}

func (f InstallForm) Error(ctx *macaron.Context, errs binding.Errors) {
	logger.Debug(errs)
	if len(errs) == 0 {
		return
	}
	json := utils.JsonResponse{}
	content := json.CommonFailure("表单验证失败, 请检测输入")

	ctx.Resp.Write([]byte(content))
}

func Create(ctx *macaron.Context) {
	if app.Installed {
		ctx.Redirect("/")
	}
	ctx.Data["Title"] = "安装"
	ctx.Data["DisableNav"] = true
	ctx.HTML(200, "install/create")
}

// 安装
func Store(ctx *macaron.Context, form InstallForm) string {
	var appConfig *setting.Setting
	var err error
	json := utils.JsonResponse{}
	if app.Installed {
		return json.CommonFailure("系统已安装!")
	}
	if app.AppConfigType == "env" {
		appConfig, err = setting.ReadEnv(app.AppConfigPrefix)
		if err != nil {
			return json.CommonFailure("读取应用配置失败", err)
		}
	} else {
		err := testDbConnection(form)
		if err != nil {
			return json.CommonFailure("数据库连接失败", err)
		}
		// 写入数据库配置
		err = writeConfig(form)
		if err != nil {
			return json.CommonFailure("数据库配置写入文件失败", err)
		}

		appConfig, err = setting.ReadIni(app.AppConfig)
		if err != nil {
			return json.CommonFailure("读取应用配置失败", err)
		}
	}
	if form.AdminPassword != form.ConfirmAdminPassword {
		return json.CommonFailure("两次输入密码不匹配")
	}

	app.Setting = appConfig

	models.Db = models.CreateDb()
	// 创建数据库表
	migration := new(models.Migration)
	err = migration.Install(form.DbName)
	if err != nil {
		return json.CommonFailure(fmt.Sprintf("创建数据库表失败-%s", err.Error()), err)
	}

	// 创建管理员账号
	err = createAdminUser(form)
	if err != nil {
		return json.CommonFailure("创建管理员账号失败", err)
	}

	// 创建安装锁
	err = app.CreateInstallLock()
	if err != nil {
		return json.CommonFailure("创建文件安装锁失败", err)
	}

	// 更新版本号文件
	app.UpdateVersionFile()

	app.Installed = true
	// 初始化定时任务
	serviceTask := new(service.Task)
	serviceTask.Initialize()

	return json.Success("安装成功", nil)
}

// 配置写入文件
func writeConfig(form InstallForm) error {
	dbConfig := []string{
		"db.engine", form.DbType,
		"db.host", form.DbHost,
		"db.port", strconv.Itoa(form.DbPort),
		"db.user", form.DbUsername,
		"db.password", form.DbPassword,
		"db.database", form.DbName,
		"db.prefix", form.DbTablePrefix,
		"db.charset", "utf8",
		"db.max.idle.conns", "30",
		"db.max.open.conns", "100",
		"allow_ips", "",
		"app.name", "定时任务管理系统", // 应用名称
		"api.key", "",
		"api.secret", "",
		"enable_tls", "false",
		"ca_file", "",
		"cert_file", "",
		"key_file", "",
	}

	return setting.Write(dbConfig, app.AppConfig)
}

// 创建管理员账号
func createAdminUser(form InstallForm) error {
	user := new(models.User)
	user.Name = form.AdminUsername
	user.Password = form.AdminPassword
	user.Email = form.AdminEmail
	user.IsAdmin = 1
	_, err := user.Create()

	return err
}

// 测试数据库连接
func testDbConnection(form InstallForm) error {
	var s setting.Setting
	s.DbEngine = form.DbType
	s.DbHost = form.DbHost
	s.DbPort = form.DbPort
	s.DbUser = form.DbUsername
	s.DbPassword = form.DbPassword
	s.DbCharset = "utf8"
	db, err := models.CreateTmpDb(&s)
	if err != nil {
		return err
	}

	defer db.Close()
	err = db.Ping()

	return err

}
