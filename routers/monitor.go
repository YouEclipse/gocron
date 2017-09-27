package routers

import (
	"gocron/modules/app"
	"net/http"

	"gopkg.in/macaron.v1"
)

// 首页
func Monitor(ctx *macaron.Context) {
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"version": app.VersionId,
	})
}
