package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	Uri            string `json:"uri"`
	Author         string `json:"author"`
	Email          string `json:"email"`
	Website        string `json:"website"`
	Content        string `json:"content"`
	PId            string `json:"pid"`
	RId            string `json:"rid"`
	TurnstileToken string `json:"turnstileToken"`
}

const turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var (
	errTurnstileNotConfigured = errors.New("Turnstile secret key is not configured")
	errTurnstileRejected      = errors.New("Turnstile rejected the token")
)

type turnstileResponse struct {
	Success bool `json:"success"`
}

func verifyTurnstile(ctx context.Context, token string) error {
	secret := os.Getenv("TURNSTILE_SECRET_KEY")
	if strings.TrimSpace(secret) == "" {
		return errTurnstileNotConfigured
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return verifyTurnstileRequest(ctx, client, turnstileSiteverifyURL, secret, token)
}

func verifyTurnstileRequest(ctx context.Context, client *http.Client, endpoint, secret, token string) error {
	form := url.Values{
		"secret":   {secret},
		"response": {token},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create Turnstile verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send Turnstile verification request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Turnstile returned HTTP %d", response.StatusCode)
	}

	var result turnstileResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return fmt.Errorf("decode Turnstile response: %w", err)
	}
	if !result.Success {
		return errTurnstileRejected
	}

	return nil
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

	if strings.TrimSpace(newComment.TurnstileToken) == "" {
		return e.BadRequestError("Turnstile token is required", nil)
	}
	if err := verifyTurnstile(e.Request.Context(), newComment.TurnstileToken); err != nil {
		switch {
		case errors.Is(err, errTurnstileNotConfigured):
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"message": "Comment verification is not configured"})
		case errors.Is(err, errTurnstileRejected):
			return e.BadRequestError("Turnstile verification failed", err)
		default:
			return e.JSON(http.StatusBadGateway, map[string]string{"message": "Comment verification is temporarily unavailable"})
		}
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
