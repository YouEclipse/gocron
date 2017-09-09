package loginlog

import (
    "gopkg.in/macaron.v1"
    "github.com/Unknwon/paginater"
    "fmt"
    "gocron/modules/logger"
    "gocron/models"
    "gocron/routers/base"
    "html/template"
)

func Index(ctx *macaron.Context)  {
    loginLogModel := new(models.LoginLog)
    params := models.CommonMap{}
    base.ParsePageAndPageSize(ctx, params)
    total, err := loginLogModel.Total()
    loginLogs, err := loginLogModel.List(params)
    if err != nil {
        logger.Error(err)
    }
    PageParams := fmt.Sprintf("page_size=%d", params["PageSize"]);
    params["PageParams"] = template.URL(PageParams)
    p := paginater.New(int(total), params["PageSize"].(int), params["Page"].(int), 5)
    ctx.Data["Pagination"] = p
    ctx.Data["Title"] = "登录日志"
    ctx.Data["LoginLogs"] = loginLogs
    ctx.Data["Params"] = params
    ctx.HTML(200, "manage/login_log")
}