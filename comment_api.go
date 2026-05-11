package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
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

	se.Router.GET("/api/unsubscribe", func(e *core.RequestEvent) error {
		return CheckUnsubscribeToken(app, e)
	})

	se.Router.POST("/api/unsubscribe", func(e *core.RequestEvent) error {
		return PostUnsubscribe(app, e)
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
			security.MD5(v.GetString("email")),
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
					security.MD5(v.GetString("email")),
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
		security.MD5(record.GetString("email")),
		record.GetString("website"),
		record.GetString("content"),
		record.GetBool("isMod"),
		record.GetString("pid"),
		record.GetString("rid"),
		[]comment{},
	}
	return e.JSON(http.StatusOK, body)
}

// CheckUnsubscribeToken 检查退订页面的token是否有效
func CheckUnsubscribeToken(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	token := e.Request.URL.Query().Get("token")
	if token == "" {
		return e.BadRequestError("Missing token", nil)
	}

	records, err := app.FindRecordsByFilter(
		"comments",
		"notify = {:token}",
		"",
		1,
		0,
		dbx.Params{"token": token},
	)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return e.NotFoundError("Token not found", nil)
	}

	return e.String(http.StatusOK, "Token is valid")
}

type unsubscribeRequest struct {
	Token  string `json:"token"`
	Action string `json:"action"`
}

type unsubscribeResult struct {
	Message string `json:"message"`
}

// PostUnsubscribe 处理退订请求
func PostUnsubscribe(app *pocketbase.PocketBase, e *core.RequestEvent) error {
	req := new(unsubscribeRequest)
	if err := e.BindBody(&req); err != nil {
		return e.BadRequestError("Failed to read request body", err)
	}
	if req.Token == "" {
		return e.BadRequestError("Missing token", nil)
	}
	if req.Action != "comment" && req.Action != "all" {
		return e.BadRequestError("Invalid action", nil)
	}

	records, err := app.FindRecordsByFilter(
		"comments",
		"notify = {:token}",
		"",
		1,
		0,
		dbx.Params{"token": req.Token},
	)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return e.NotFoundError("Token not found", nil)
	}

	record := records[0]
	if req.Action == "comment" {
		record.Set("notify", "")
		if err := app.Save(record); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, unsubscribeResult{Message: "已取消此评论的后续回复通知"})
	}

	email := record.GetString("email")
	if email == "" {
		return e.BadRequestError("Record email missing", nil)
	}

	records, err = app.FindRecordsByFilter(
		"comments",
		"email = {:email} && notify != ''",
		"",
		1000,
		0,
		dbx.Params{"email": email},
	)
	if err != nil {
		return err
	}

	count := 0
	for _, r := range records {
		r.Set("notify", "")
		if err := app.Save(r); err != nil {
			return err
		}
		count++
	}

	return e.JSON(http.StatusOK, unsubscribeResult{Message: fmt.Sprintf("已取消%d条评论的后续回复通知", count)})
}
