package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type comment struct {
	Id      string    `json:"id"`
	Created string    `json:"created"`
	Author  string    `json:"author"`
	Avatar  string    `json:"avatar"`
	Website string    `json:"website"`
	Content string    `json:"content"`
	IsMod   bool      `json:"is_mod"`
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

	// app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// serves static files from the provided public dir (if exists)
		se.Router.GET("/{path...}", apis.Static(os.DirFS("./pb_public"), true))

		// get comments of a page
		se.Router.GET("/api/comment/", func(e *core.RequestEvent) error {
			uri := e.Request.URL.Query().Get("uri")
			commentList := []comment{}

			records, err := app.FindAllRecords("comments",
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
					v.GetBool("is_mod"),
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
			return e.JSON(http.StatusOK, body)
		})

		// handle new comment
		se.Router.POST("/api/comment/", func(e *core.RequestEvent) error {

			newComment := new(newComment)
			if err := e.BindBody(&newComment); err != nil {
				return e.BadRequestError("Failed to read request body", err)
			}

			collection, err := app.FindCollectionByNameOrId("comments")
			if err != nil {
				return err
			}

			record := core.NewRecord(collection)
			record.Load(map[string]any{
				"uri":     newComment.Uri,
				"author":  newComment.Author,
				"email":   newComment.Email,
				"website": newComment.Website,
				"content": newComment.Content,
				"parent":  newComment.Parent,
			})

			err = app.Save(record)
			if err != nil {
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
				record.GetBool("is_mod"),
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

			return e.JSON(http.StatusOK, body)
		})

		return se.Next()
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
