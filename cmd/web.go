package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"gocron/models"
	"gocron/modules/app"
	"gocron/modules/logger"
	"gocron/modules/rpc/grpcpool"
	"gocron/modules/setting"
	"gocron/routers"
	"gocron/service"

	"github.com/urfave/cli"
	"gopkg.in/macaron.v1"
)

// web服务器默认端口
const DefaultPort = 5920

var CmdWeb = cli.Command{
	Name:   "web",
	Usage:  "run web server",
	Action: runWeb,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "0.0.0.0",
			Usage: "bind host",
		},
		cli.IntFlag{
			Name:  "port,p",
			Value: DefaultPort,
			Usage: "bind port",
		},
		cli.StringFlag{
			Name:  "env,e",
			Value: "prod",
			Usage: "runtime environment, dev|test|prod",
		},
		cli.StringFlag{
			Name:  "config,c",
			Value: "ini",
			Usage: "config type , ini|env",
		},
		cli.StringFlag{
			Name:  "prefix,pre",
			Value: "cron",
			Usage: "envconfig prefix ",
		},
	},
}

func runWeb(ctx *cli.Context) {
	// 设置运行环境
	setEnvironment(ctx)
	// 初始化应用
	cfgType := parseConfigType(ctx)
	cfgPrefx := parseConfigPrefix(ctx)
	app.InitEnv(ctx.App.Version, cfgType, cfgPrefx)
	// 初始化模块 DB、定时任务等
	initModule()
	// 捕捉信号,配置热更新等
	go catchSignal()
	m := macaron.Classic()

	// 注册路由
	routers.Register(m)
	// 注册中间件.
	routers.RegisterMiddleware(m)
	host := parseHost(ctx)
	port := parsePort(ctx)
	m.Run(host, port)
}

func initModule() {
	if !app.Installed {
		return
	}
	var config = &setting.Setting{}
	var err error

	switch app.AppConfigType {
	case "env":
		config, err = setting.ReadEnv(app.AppConfigPrefix)
		if err != nil {
			logger.Fatal("读取应用配置失败", err)
		}
	case "ini":
		config, err = setting.ReadIni(app.ConfDir)
		if err != nil {
			logger.Fatal("读取应用配置失败", err)
		}
	default:
		config, err = setting.ReadIni(app.ConfDir)
		if err != nil {
			logger.Fatal("读取应用配置失败", err)
		}
	}

	app.Setting = config

	// 初始化DB
	models.Db = models.CreateDb()

	// 版本升级
	upgradeIfNeed()

	// 初始化定时任务
	serviceTask := new(service.Task)
	serviceTask.Initialize()
}

// 解析端口
func parsePort(ctx *cli.Context) int {
	var port int = DefaultPort
	if ctx.IsSet("port") {
		port = ctx.Int("port")
	}
	if port <= 0 || port >= 65535 {
		port = DefaultPort
	}

	return port
}

func parseHost(ctx *cli.Context) string {
	if ctx.IsSet("host") {
		return ctx.String("host")
	}

	return "0.0.0.0"
}

func parseConfigType(ctx *cli.Context) string {
	if ctx.IsSet("config") {
		return ctx.String("config")
	}
	return "ini"
}

func parseConfigPrefix(ctx *cli.Context) string {
	if ctx.IsSet("prefix") {
		return ctx.String("prefix")
	}
	return "envconfig"
}

func setEnvironment(ctx *cli.Context) {
	var env string = "prod"
	if ctx.IsSet("env") {
		env = ctx.String("env")
	}

	switch env {
	case "test":
		macaron.Env = macaron.TEST
	case "dev":
		macaron.Env = macaron.DEV
	default:
		macaron.Env = macaron.PROD
	}
}

// 捕捉信号
func catchSignal() {
	c := make(chan os.Signal)
	// todo 配置热更新, windows 不支持 syscall.SIGUSR1, syscall.SIGUSR2
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	for {
		s := <-c
		logger.Info("收到信号 -- ", s)
		switch s {
		case syscall.SIGHUP:
			logger.Info("收到终端断开信号, 忽略")
		case syscall.SIGINT, syscall.SIGTERM:
			shutdown()
		}
	}
}

// 应用退出
func shutdown() {
	defer func() {
		logger.Info("已退出")
		os.Exit(0)
	}()

	if !app.Installed {
		return
	}
	logger.Info("应用准备退出")
	serviceTask := new(service.Task)
	// 停止所有任务调度
	logger.Info("停止定时任务调度")
	serviceTask.StopAll()

	taskNumInRunning := service.TaskNum.Num()
	logger.Infof("正在运行的任务有%d个", taskNumInRunning)
	if taskNumInRunning > 0 {
		logger.Info("等待所有任务执行完成后退出")
	}
	for {
		if taskNumInRunning <= 0 {
			break
		}
		time.Sleep(3 * time.Second)
		taskNumInRunning = service.TaskNum.Num()
	}

	// 释放gRPC连接池
	grpcpool.Pool.ReleaseAll()
}

// 判断应用是否需要升级, 当存在版本号文件且版本小于app.VersionId时升级
func upgradeIfNeed() {
	currentVersionId := app.GetCurrentVersionId()
	// 没有版本号文件
	if currentVersionId == 0 {
		return
	}
	if currentVersionId >= app.VersionId {
		return
	}

	migration := new(models.Migration)
	logger.Infof("版本升级开始, 当前版本号%d", currentVersionId)

	migration.Upgrade(currentVersionId)
	app.UpdateVersionFile()

	logger.Infof("已升级到最新版本%d", app.VersionId)
}
