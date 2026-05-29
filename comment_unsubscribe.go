package main

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
)

// SetupUnsubscribeRoute 设置邮件通知退订相关路由
func SetupUnsubscribeRoute(app *pocketbase.PocketBase, se *core.ServeEvent) {

	se.Router.GET("/unsubscribe", func(e *core.RequestEvent) error {
		return GetUnsubscribe(app, e)
	})
}

// GetUnsubscribe 处理退订页面的请求
func GetUnsubscribe(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	token := e.Request.URL.Query().Get("token")
	action := e.Request.URL.Query().Get("action")

	// 验证 token
	records, err := app.FindRecordsByFilter(
		"comments",
		"notify = {:token}",
		"",
		1,
		0,
		dbx.Params{"token": token},
	)
	if err != nil {
		data := map[string]any{
			"Error": "发生错误，请稍后重试",
		}
		return renderUnsubscribeTemplate(e, data)
	}
	if len(records) == 0 {
		data := map[string]any{
			"Error": "无效的退订令牌，请使用邮件中的退订链接。",
		}
		return renderUnsubscribeTemplate(e, data)
	}

	// 如果有 action 参数，执行退订操作
	if action == "comment" {
		record := records[0]
		record.Set("notify", "")
		if err := app.Save(record); err != nil {
			data := map[string]any{
				"Error": "退订失败，请稍后重试",
			}
			return renderUnsubscribeTemplate(e, data)
		}
		data := map[string]any{
			"Success": true,
			"Message": "已取消此评论的后续回复通知",
		}
		return renderUnsubscribeTemplate(e, data)
	}

	if action == "all" {
		record := records[0]
		email := record.GetString("email")
		if email == "" {
			data := map[string]any{
				"Error": "退订失败，请稍后重试",
			}
			return renderUnsubscribeTemplate(e, data)
		}

		allRecords, err := app.FindRecordsByFilter(
			"comments",
			"email = {:email} && notify != ''",
			"",
			1000,
			0,
			dbx.Params{"email": email},
		)
		if err != nil {
			data := map[string]any{
				"Error": "退订失败，请稍后重试",
			}
			return renderUnsubscribeTemplate(e, data)
		}

		count := 0
		for _, r := range allRecords {
			r.Set("notify", "")
			if err := app.Save(r); err != nil {
				continue
			}
			count++
		}

		message := fmt.Sprintf("已取消%d条评论的后续回复通知", count)
		data := map[string]any{
			"Success": true,
			"Message": message,
		}
		return renderUnsubscribeTemplate(e, data)
	}

	// 没有 action 参数，显示选择页面
	data := map[string]any{
		"Token": token,
	}
	return renderUnsubscribeTemplate(e, data)
}

// renderUnsubscribeTemplate 渲染退订模板
func renderUnsubscribeTemplate(e *core.RequestEvent, data map[string]any) error {
	html, err := template.NewRegistry().LoadFiles("views/unsubscribe.html").Render(data)
	if err != nil {
		return e.InternalServerError("Failed to render template", err)
	}
	return e.HTML(http.StatusOK, html)
}
