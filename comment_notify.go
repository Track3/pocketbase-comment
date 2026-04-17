package main

import (
	"net/mail"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/template"
)

func SetupCommentNotification(app *pocketbase.PocketBase) {

	app.OnRecordAfterCreateSuccess("comments").BindFunc(func(e *core.RecordEvent) error {

		adminEmail := os.Getenv("COMMENT_ADMIN_EMAIL")
		isMod := e.Record.GetBool("isMod")
		data := map[string]string{
			"siteName": os.Getenv("SITE_NAME"),
			"siteURL":  os.Getenv("SITE_URL"),
			"id":       e.Record.Id,
			"uri":      e.Record.GetString("uri"),
			"author":   e.Record.GetString("author"),
			"content":  e.Record.GetString("content"),
		}

		//通知管理员
		if isMod == false {
			err := notifyAdmin(e, data, adminEmail)
			if err != nil {
				return err
			}
		}

		//通知原评论作者
		rid := e.Record.GetString("rid")
		if rid != "" {
			opRecord, err := e.App.FindRecordById("comments", rid)
			if err != nil {
				return err
			}
			if (isMod == false) && (opRecord.GetString("notify") != "") {
				opEmail := opRecord.GetString("email")
				data["opAuthor"] = opRecord.GetString("author")
				data["opContent"] = opRecord.GetString("content")
				err := notifyOP(e, data, opEmail)
				if err != nil {
					return err
				}
			}
		}

		return e.Next()
	})
}

func notifyAdmin(e *core.RecordEvent, data map[string]string, adminEmail string) error {
	htmlToAdmin, err := template.NewRegistry().LoadFiles("views/notify_admin.html").Render(data)
	if err != nil {
		return err
	}
	go func() {
		e.App.NewMailClient().Send(&mailer.Message{
			From: mail.Address{
				Address: e.App.Settings().Meta.SenderAddress,
				Name:    e.App.Settings().Meta.SenderName,
			},
			To:      []mail.Address{{Address: adminEmail}},
			Subject: "新评论通知",
			HTML:    htmlToAdmin,
		})
	}()
	return nil
}

func notifyOP(e *core.RecordEvent, data map[string]string, opEmail string) error {
	htmlToOP, err := template.NewRegistry().LoadFiles("views/notify_op.html").Render(data)
	if err != nil {
		return err
	}
	go func() {
		e.App.NewMailClient().Send(&mailer.Message{
			From: mail.Address{
				Address: e.App.Settings().Meta.SenderAddress,
				Name:    e.App.Settings().Meta.SenderName,
			},
			To:      []mail.Address{{Address: opEmail}},
			Subject: "新评论回复通知",
			HTML:    htmlToOP,
		})
	}()
	return nil
}
