package task

import (
    "gopkg.in/macaron.v1"
    "gocron/models"
    "gocron/modules/logger"
    "gocron/modules/utils"
    "gocron/service"
    "strconv"
    "github.com/jakecoffman/cron"
    "github.com/Unknwon/paginater"
    "fmt"
    "html/template"
    "gocron/routers/base"
    "github.com/go-macaron/binding"
    "strings"
)

type TaskForm struct {
    Id int
    Level models.TaskLevel `binding:"Required;In(1,2)"`
    DependencyStatus models.TaskDependencyStatus
    DependencyTaskId string
    Name string `binding:"Required;MaxSize(32)"`
    Spec string
    Protocol models.TaskProtocol `binding:"In(1,2)"`
    Command string `binding:"Required;MaxSize(256)"`
    Timeout int `binding:"Range(0,86400)"`
    Multi  int8 `binding:"In(1,2)"`
    RetryTimes int8
    HostId string
    Tag string
    Remark string
    NotifyStatus int8 `binding:"In(1,2,3)"`
    NotifyType int8 `binding:"In(1,2,3)"`
    NotifyReceiverId string
}


func (f TaskForm) Error(ctx *macaron.Context, errs binding.Errors) {
    if len(errs) == 0 {
        return
    }
    json := utils.JsonResponse{}
    content := json.CommonFailure("表单验证失败, 请检测输入")

    ctx.Resp.Write([]byte(content))
}

// 首页
func Index(ctx *macaron.Context)  {
    taskModel := new(models.Task)
    queryParams := parseQueryParams(ctx)
    total, err := taskModel.Total(queryParams)
    if err != nil {
        logger.Error(err)
    }
    tasks, err := taskModel.List(queryParams)
    if err != nil {
        logger.Error(err)
    }
    name, ok := queryParams["name"].(string)
    var safeNameHTML = ""
    if ok {
        safeNameHTML = template.HTMLEscapeString(name)
    }
    PageParams := fmt.Sprintf("id=%d&host_id=%d&name=%s&protocol=%d&tag=%s&status=%d&page_size=%d",
        queryParams["Id"], queryParams["HostId"], safeNameHTML, queryParams["Protocol"], queryParams["Tag"], queryParams["Status"], queryParams["PageSize"]);
    queryParams["PageParams"] = template.URL(PageParams)
    p := paginater.New(int(total), queryParams["PageSize"].(int), queryParams["Page"].(int), 5)
    ctx.Data["Pagination"] = p
    setHostsToTemplate(ctx)
    ctx.Data["Params"] = queryParams
    ctx.Data["Title"] = "任务列表"
    ctx.Data["Tasks"] = tasks
    ctx.HTML(200, "task/index")
}

// 新增页面
func Create(ctx *macaron.Context)  {
    setHostsToTemplate(ctx)
    ctx.Data["Title"] = "添加任务"
    ctx.HTML(200, "task/task_form")
}

// 编辑页面
func Edit(ctx *macaron.Context)  {
    id := ctx.ParamsInt(":id")
    taskModel := new(models.Task)
    task, err := taskModel.Detail(id)
    if err != nil || task.Id != id {
        logger.Errorf("编辑任务#获取任务详情失败#任务ID-%d#%s", id, err.Error())
        ctx.Redirect("/task")
    }
    hostModel := new(models.Host)
    hostModel.PageSize = -1
    hosts, err := hostModel.List(models.CommonMap{})
    if err != nil {
        logger.Error(err)
    } else {
        for i, host := range(hosts) {
            if inHosts(task.Hosts, host.Id) {
                hosts[i].Selected = true
            }
        }
    }

    ctx.Data["Task"]  = task
    ctx.Data["Hosts"] = hosts
    ctx.Data["Title"] = "编辑"
    ctx.HTML(200, "task/task_form")
}

// 保存任务
func Store(ctx *macaron.Context, form TaskForm) string  {
    json := utils.JsonResponse{}
    taskModel := models.Task{}
    var id int = form.Id
    nameExists, err := taskModel.NameExist(form.Name, form.Id)
    if err != nil {
        return json.CommonFailure(utils.FailureContent, err)
    }
    if nameExists {
        return json.CommonFailure("任务名称已存在")
    }

    if form.Protocol == models.TaskRPC && form.HostId == "" {
        return json.CommonFailure("请选择主机名")
    }

    taskModel.Name = form.Name
    taskModel.Protocol = form.Protocol
    taskModel.Command = form.Command
    taskModel.Timeout = form.Timeout
    taskModel.Tag = form.Tag
    taskModel.Remark = form.Remark
    taskModel.Multi = form.Multi
    taskModel.RetryTimes = form.RetryTimes
    if taskModel.Multi != 1 {
        taskModel.Multi = 0
    }
    taskModel.NotifyStatus = form.NotifyStatus - 1
    taskModel.NotifyType = form.NotifyType - 1
    taskModel.NotifyReceiverId = form.NotifyReceiverId
    taskModel.Spec = form.Spec
    taskModel.Level = form.Level
    taskModel.DependencyStatus = form.DependencyStatus
    taskModel.DependencyTaskId = strings.TrimSpace(form.DependencyTaskId)
    if taskModel.NotifyStatus > 0 && taskModel.NotifyReceiverId == "" {
        return json.CommonFailure("至少选择一个通知接收者")
    }
    if taskModel.Protocol == models.TaskHTTP {
        command := strings.ToLower(taskModel.Command)
        if !strings.HasPrefix(command, "http://") && !strings.HasPrefix(command, "https://") {
            return json.CommonFailure("请输入正确的URL地址")
        }
        if taskModel.Timeout > 300 {
            return json.CommonFailure("HTTP任务超时时间不能超过300秒")
        }
    }

    if taskModel.RetryTimes > 10 || taskModel.RetryTimes < 0 {
        return json.CommonFailure("任务重试次数取值0-10")
    }

    if (taskModel.DependencyStatus != models.TaskDependencyStatusStrong &&
        taskModel.DependencyStatus != models.TaskDependencyStatusWeak) {
        return json.CommonFailure("请选择依赖关系")
    }

    if taskModel.Level == models.TaskLevelParent {
        _, err = cron.Parse(form.Spec)
        if err != nil {
            return json.CommonFailure("crontab表达式解析失败", err)
        }
    } else {
        taskModel.DependencyTaskId = ""
        taskModel.Spec = ""
    }

    if id > 0 && taskModel.DependencyTaskId != "" {
        dependencyTaskIds := strings.Split(taskModel.DependencyTaskId, ",")
        if utils.InStringSlice(dependencyTaskIds, strconv.Itoa(id)) {
            return json.CommonFailure("不允许设置当前任务为子任务")
        }
    }

    if id == 0 {
        // 任务添加后开始调度执行
        taskModel.Status = models.Running
        id, err = taskModel.Create()
    } else {
        _, err = taskModel.UpdateBean(id)
    }

    if err != nil {
        return json.CommonFailure("保存失败", err)
    }

    taskHostModel := new(models.TaskHost)
    if form.Protocol == models.TaskRPC {
        hostIdStrList := strings.Split(form.HostId, ",")
        hostIds := make([]int, len(hostIdStrList))
        for i, hostIdStr := range hostIdStrList {
            hostIds[i], _ = strconv.Atoi(hostIdStr)
        }
        taskHostModel.Add(id, hostIds)
    } else {
        taskHostModel.Remove(id)
    }

    status, err := taskModel.GetStatus(id)
    if status == models.Enabled && taskModel.Level == models.TaskLevelParent {
        addTaskToTimer(id)
    }

    return json.Success("保存成功", nil)
}

// 删除任务
func Remove(ctx *macaron.Context) string {
    id  := ctx.ParamsInt(":id")
    json := utils.JsonResponse{}
    taskModel := new(models.Task)
    _, err := taskModel.Delete(id)
    if err != nil {
        return json.CommonFailure(utils.FailureContent, err)
    }

    taskHostModel := new(models.TaskHost)
    taskHostModel.Remove(id)

    service.Cron.RemoveJob(strconv.Itoa(id))

    return json.Success(utils.SuccessContent, nil)
}

// 激活任务
func Enable(ctx *macaron.Context) string {
    return changeStatus(ctx, models.Enabled)
}

// 暂停任务
func Disable(ctx *macaron.Context) string {
    return changeStatus(ctx, models.Disabled)
}

// 手动运行任务
func Run(ctx *macaron.Context) string {
    id := ctx.ParamsInt(":id")
    json := utils.JsonResponse{}
    taskModel := new(models.Task)
    task , err := taskModel.Detail(id)
    if err != nil || task.Id <= 0 {
        return json.CommonFailure("获取任务详情失败", err)
    }

    task.Spec = "手动运行"
    serviceTask := new(service.Task)
    serviceTask.Run(task)

    return json.Success("任务已开始运行, 请到任务日志中查看结果", nil);
}

// 改变任务状态
func changeStatus(ctx *macaron.Context, status models.Status) string {
    id  := ctx.ParamsInt(":id")
    json := utils.JsonResponse{}
    taskModel := new(models.Task)
    _, err := taskModel.Update(id, models.CommonMap{
        "Status": status,
    })
    if err != nil {
        return json.CommonFailure(utils.FailureContent, err)
    }

    if status == models.Enabled {
        addTaskToTimer(id)
    } else {
        service.Cron.RemoveJob(strconv.Itoa(id))
    }

    return json.Success(utils.SuccessContent, nil)
}

// 添加任务到定时器
func addTaskToTimer(id int)  {
    taskModel := new(models.Task)
    task, err := taskModel.Detail(id)
    if err != nil {
        logger.Error(err)
        return
    }

    taskService := service.Task{}
    taskService.Add(task)
}

// 解析查询参数
func parseQueryParams(ctx *macaron.Context) (models.CommonMap) {
    var params models.CommonMap = models.CommonMap{}
    params["Id"] = ctx.QueryInt("id")
    params["HostId"] = ctx.QueryInt("host_id")
    params["Name"] = ctx.QueryTrim("name")
    params["Protocol"] = ctx.QueryInt("protocol")
    params["Tag"] = ctx.QueryTrim("tag")
    status := ctx.QueryInt("status")
    if status >=0 {
        status -= 1
    }
    params["Status"] = status
    base.ParsePageAndPageSize(ctx, params)

    return params
}

func setHostsToTemplate(ctx *macaron.Context)  {
    hostModel := new(models.Host)
    hostModel.PageSize = -1
    hosts, err := hostModel.List(models.CommonMap{})
    if err != nil {
        logger.Error(err)
    }
    ctx.Data["Hosts"] = hosts
}

func inHosts(slice []models.TaskHostDetail, element int16) bool {
    for _, v := range slice {
        if v.HostId == element {
            return true
        }
    }

    return false
}