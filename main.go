package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/forms"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type comment struct {
	Id      string    `json:"id"`
	Created string    `json:"created"`
	Author  string    `json:"author"`
	Avatar  string    `json:"avatar"`
	Website string    `json:"website"`
	Content string    `json:"content"`
	Reply   []comment `json:"reply"`
}

type newComment struct {
	Uri     string `json:"uri"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Website string `json:"website"`
	Content string `json:"content"`
	Parent  string `json:"parent"`
}

func main() {
	app := pocketbase.New()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {

		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), true))

		// get comments of a page
		e.Router.GET("api/comment", func(c echo.Context) error {
			uri := c.QueryParam("uri")
			commentList := []comment{}

			records, err := app.Dao().FindRecordsByExpr("comments",
				dbx.HashExp{"uri": uri},
			)
			if err != nil {
				return err
			}

			for _, v := range records {
				emailHash := calcMD5(v.GetString("email"))
				entry := comment{
					v.Id,
					v.GetString("created"),
					v.GetString("author"),
					emailHash,
					v.GetString("website"),
					v.GetString("content"),
					[]comment{},
				}

				if v.GetString("parent") == "" {
					commentList = append(commentList, entry)
				} else {
					for i := range commentList {
						if commentList[i].Id == v.GetString("parent") {
							commentList[i].Reply = append(commentList[i].Reply, entry)
							break
						}
					}
				}
			}

			body := struct {
				Count int       `json:"count"`
				List  []comment `json:"list"`
			}{len(records), commentList}
			return c.JSON(200, body)
		},
			apis.ActivityLogger(app),
		)

		// handle new comment
		e.Router.POST("api/comment", func(c echo.Context) error {

			newComment := new(newComment)
			if err := c.Bind(newComment); err != nil {
				return c.String(http.StatusBadRequest, "bad request")
			}

			collection, err := app.Dao().FindCollectionByNameOrId("comments")
			if err != nil {
				return err
			}
			record := models.NewRecord(collection)
			form := forms.NewRecordUpsert(app, record)

			form.LoadData(map[string]any{
				"uri":     newComment.Uri,
				"author":  newComment.Author,
				"email":   newComment.Email,
				"website": newComment.Website,
				"content": newComment.Content,
				"parent":  newComment.Parent,
			})

			// validate and submit (internally it calls app.Dao().SaveRecord(record) in a transaction)
			if err := form.Submit(); err != nil {
				return err
			}

			emailHash := calcMD5(record.GetString("email"))
			body := comment{
				record.Id,
				record.GetString("created"),
				record.GetString("author"),
				emailHash,
				record.GetString("website"),
				record.GetString("content"),
				[]comment{},
			}

			// send email notification if COMMENT_NOTIFY_EMAIL is set
			COMMENT_NOTIFY_EMAIL := os.Getenv("COMMENT_NOTIFY_EMAIL")
			if COMMENT_NOTIFY_EMAIL != "" {
				message := &mailer.Message{
					From: mail.Address{
						Address: app.Settings().Meta.SenderAddress,
						Name:    app.Settings().Meta.SenderName,
					},
					To:      []mail.Address{{Address: COMMENT_NOTIFY_EMAIL}},
					Subject: "📮新评论通知",
					HTML:    "<p><b>" + body.Author + "</b> 说:</p><p>" + body.Content + "</p>",
				}
				go func() {
					app.NewMailClient().Send(message)
				}()
			}

			return c.JSON(200, body)
		},
			apis.ActivityLogger(app),
		)

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// calc email hash helper
func calcMD5(input string) string {
	data := []byte(input)
	return fmt.Sprintf("%x", md5.Sum(data))
}
