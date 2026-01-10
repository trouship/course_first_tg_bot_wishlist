package telegram

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"tg_game_wishlist/lib/e"
	"time"
)

type Client struct {
	host     string
	basePath string
	client   http.Client
}

const (
	getUpdatesMethod     = "getUpdates"
	sendMessageMethod    = "sendMessage"
	answerCallbackMethod = "answerCallbackQuery"
)

func New(host string, token string, timeout int) *Client {
	return &Client{
		host:     host,
		basePath: newBasePath(token),
		client: http.Client{
			Timeout: 65 * time.Second,
		},
	}
}

func newBasePath(token string) string {
	return "bot" + token
}

func (c *Client) Updates(ctx context.Context, offset int, limit int, timeout int) (updates []Update, err error) {
	defer func() { err = e.WrapIfNil("can't get updates", err) }()

	q := url.Values{}
	q.Add("offset", strconv.Itoa(offset))
	q.Add("limit", strconv.Itoa(limit))
	q.Add("timeout", strconv.Itoa(timeout))

	data, err := c.doRequest(ctx, getUpdatesMethod, http.MethodGet, q)
	if err != nil {
		return nil, err
	}

	var res UpdatesResponse

	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return res.Result, nil
}

func (c *Client) SendMessage(ctx context.Context, chatId int, text string) error {
	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatId))
	q.Add("text", text)

	_, err := c.doRequest(ctx, sendMessageMethod, http.MethodGet, q)
	if err != nil {
		return e.Wrap("can't send message", err)
	}

	return nil
}

func (c *Client) SendMessageWithKeyboard(ctx context.Context, chatId int, text string, keyboard *InlineKeyboardMarkup) error {
	jsonKeyboard, err := json.Marshal(keyboard)
	if err != nil {
		return e.Wrap("can't marshal inline keyboard in message", err)
	}

	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatId))
	q.Add("text", text)
	q.Add("reply_markup", string(jsonKeyboard))

	_, err = c.doRequest(ctx, sendMessageMethod, http.MethodPost, q)
	if err != nil {
		return e.Wrap("can't send message with inline keyboard", err)
	}

	return nil
}

func (c *Client) AnswerCallBack(ctx context.Context, callbackId string, text string, showAlert bool) error {
	q := url.Values{}
	q.Add("callback_query_id", callbackId)
	q.Add("text", text)
	q.Add("show_alert", strconv.FormatBool(showAlert))

	_, err := c.doRequest(ctx, answerCallbackMethod, http.MethodPost, q)
	if err != nil {
		return e.Wrap("can't answer callback", err)
	}

	return nil

}

func (c *Client) doRequest(ctx context.Context, method string, httpMethod string, q url.Values) (data []byte, err error) {
	defer func() { err = e.WrapIfNil("can't do get request", err) }()

	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join(c.basePath, method),
	}

	log.Print(u.String())
	req, err := http.NewRequestWithContext(ctx, httpMethod, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
