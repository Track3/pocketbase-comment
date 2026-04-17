package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// 提供静态文件
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), true))

		// 设置评论API
		SetupCommentAPI(app, se)

		return se.Next()
	})

	// 评论邮件通知
	if os.Getenv("COMMENT_NOTIFY_ENABLED") == "true" {
		SetupCommentNotification(app)
	}

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
