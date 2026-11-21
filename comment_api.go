package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ganigeorgiev.com/pocketbase/dbx"
	"ganigeorgiev.com/pocketbase/pocketbase"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	"ganigeorgiev.com/pocketbase/pocketbase/tools/security"
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
	Uri           string `json:"uri"`
	Author        string `json:"author"`
	Email         string `json:"email"`
	Website       string `json:"website"`
	Content       string `json:"content"`
	PId           string `json:"pid"`
	RId           string `json:"rid"`
	FCaptchaToken string `json:"fcaptchaToken"`
}

const (
	defaultFCaptchaURL = "https://fcaptcha.zak.ee"
)

var (
	errFCaptchaNotConfigured = errors.New("FCaptcha verify secret is not configured")
	errFCaptchaRejected      = errors.New("FCaptcha rejected the token")
)

type fcaptchaVerifyRequest struct {
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

type fcaptchaVerifyResponse struct {
	Valid     bool    `json:"valid"`
	Reason    string  `json:"reason"`
	SiteKey   string  `json:"site_key"`
	Score     float64 `json:"score"`
	Action    string  `json:"action"`
	Hostname  string  `json:"hostname"`
	Timestamp int64   `json:"timestamp"`
	CData     string  `json:"cdata"`
}

func fcaptchaVerifyURL() string {
	serverURL := strings.TrimSpace(os.Getenv("FCAPTCHA_URL"))
	if serverURL == "" {
		serverURL = defaultFCaptchaURL
	}
	return strings.TrimRight(serverURL, "/") + "/api/token/verify"
}

func verifyFCaptcha(ctx context.Context, token string) error {
	secret := strings.TrimSpace(os.Getenv("FCAPTCHA_VERIFY_SECRET"))
	if secret == "" {
		// FCaptcha falls back to FCAPTCHA_SECRET when no separate verify
		// credential is configured, so do the same here.
		secret = strings.TrimSpace(os.Getenv("FCAPTCHA_SECRET"))
	}
	if secret == "" {
		return errFCaptchaNotConfigured
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return verifyFCaptchaRequest(ctx, client, fcaptchaVerifyURL(), secret, token)
}

func verifyFCaptchaRequest(ctx context.Context, client *http.Client, endpoint, secret, token string) error {
	payload, err := json.Marshal(fcaptchaVerifyRequest{Token: token, Secret: secret})
	if err != nil {
		return fmt.Errorf("encode FCaptcha verification request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create FCaptcha verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send FCaptcha verification request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read FCaptcha response: %w", err)
	}

	// 401 means the configured verify secret is wrong, not that the visitor's
	// token is bad, so surface it as a server-side failure.
	if response.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("FCaptcha rejected the verify secret: %s", strings.TrimSpace(string(body)))
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("FCaptcha returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var result fcaptchaVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode FCaptcha response: %w", err)
	}
	if !result.Valid {
		return errFCaptchaRejected
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

	if strings.TrimSpace(newComment.FCaptchaToken) == "" {
		return e.BadRequestError("FCaptcha token is required", nil)
	}
	if err := verifyFCaptcha(e.Request.Context(), newComment.FCaptchaToken); err != nil {
		switch {
		case errors.Is(err, errFCaptchaNotConfigured):
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"message": "Comment verification is not configured"})
		case errors.Is(err, errFCaptchaRejected):
			return e.BadRequestError("FCaptcha verification failed", err)
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
