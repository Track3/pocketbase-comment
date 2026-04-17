package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type comment struct {
	Id      string    `json:"id"`
	Created string    `json:"created"`
	Author  string    `json:"author"`
	Avatar  string    `json:"avatar"`
	Website string    `json:"website,omitempty"`
	Content string    `json:"content"`
	IsMod   bool      `json:"isMod"`
	PId     string    `json:"pid,omitempty"`
	RId     string    `json:"rid,omitempty"`
	Replies []comment `json:"replies"`
}

type newComment struct {
	Uri     string `json:"uri"`
	Author  string `json:"author"`
	Email   string `json:"email"`
	Website string `json:"website"`
	Content string `json:"content"`
	PId     string `json:"pid"`
	RId     string `json:"rid"`
}

// SetupCommentAPI 设置评论相关的路由
func SetupCommentAPI(app *pocketbase.PocketBase, se *core.ServeEvent) {

	se.Router.GET("/api/comment", func(e *core.RequestEvent) error {
		return GetComment(app, e)
	})

	se.Router.POST("/api/comment", func(e *core.RequestEvent) error {
		return PostComment(app, e)
	})
}

// GetComments 获取指定URI的评论列表
func GetComment(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	uri := e.Request.URL.Query().Get("uri")
	pageStr := e.Request.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}

	commentsPerPage := 10
	offset := (page - 1) * commentsPerPage
	comments := []comment{}

	count, err := app.CountRecords("comments", dbx.HashExp{"uri": uri})
	records, err := app.FindRecordsByFilter(
		"comments",
		"uri = {:uri} && pid = ''",
		"-created",
		commentsPerPage,
		offset,
		dbx.Params{"uri": uri},
	)
	if err != nil {
		return err
	}

	errs := app.ExpandRecords(records, []string{"comments_via_pid"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("Failed to expand: %v", errs)
	}

	for _, v := range records {
		item := comment{
			v.Id,
			v.GetString("created"),
			v.GetString("author"),
			calcMD5(v.GetString("email")),
			v.GetString("website"),
			v.GetString("content"),
			v.GetBool("isMod"),
			v.GetString("pid"),
			v.GetString("rid"),
			[]comment{},
		}
		replies := v.ExpandedAll("comments_via_pid")
		if len(replies) > 0 {
			for _, v := range replies {
				replyItem := comment{
					v.Id,
					v.GetString("created"),
					v.GetString("author"),
					calcMD5(v.GetString("email")),
					v.GetString("website"),
					v.GetString("content"),
					v.GetBool("isMod"),
					v.GetString("pid"),
					v.GetString("rid"),
					[]comment{},
				}
				item.Replies = append(item.Replies, replyItem)
			}
		}
		comments = append(comments, item)
	}

	body := struct {
		Uri             string    `json:"uri"`
		Page            int       `json:"page"`
		CommentsPerPage int       `json:"commentsPerPage"`
		Count           int64     `json:"count"`
		Comments        []comment `json:"comments"`
	}{uri, page, commentsPerPage, count, comments}
	return e.JSON(http.StatusOK, body)
}

// PostComment 处理新评论提交
func PostComment(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	newComment := new(newComment)
	if err := e.BindBody(&newComment); err != nil {
		return e.BadRequestError("Failed to read request body", err)
	}

	collection, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		return err
	}

	isMod := newComment.Email == os.Getenv("COMMENT_ADMIN_EMAIL")
	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"uri":     newComment.Uri,
		"author":  newComment.Author,
		"email":   newComment.Email,
		"website": newComment.Website,
		"content": newComment.Content,
		"pid":     newComment.PId,
		"rid":     newComment.RId,
		"isMod":   isMod,
	})

	err = app.Save(record)
	if err != nil {
		return err
	}

	body := comment{
		record.Id,
		record.GetString("created"),
		record.GetString("author"),
		calcMD5(record.GetString("email")),
		record.GetString("website"),
		record.GetString("content"),
		record.GetBool("isMod"),
		record.GetString("pid"),
		record.GetString("rid"),
		[]comment{},
	}
	return e.JSON(http.StatusOK, body)
}

// calcMD5 计算字符串的MD5哈希值
func calcMD5(input string) string {
	data := []byte(input)
	return fmt.Sprintf("%x", md5.Sum(data))
}
