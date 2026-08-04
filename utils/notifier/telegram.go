package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const tgMaxMsgLen = 4096

type telegramState struct {
	token          string
	chatID         int64
	httpClient     *http.Client
	parseMode      string
	disablePreview bool
	enabled        bool
}

var tg atomic.Value // telegramState

// TelegramInit инициализирует Telegram-отправитель.
// Можно добавить опции через WithTelegram*.
func TelegramInit(token string, chatID int64, opts ...TelegramOption) {
	st := telegramState{
		token:          token,
		chatID:         chatID,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		parseMode:      "",
		disablePreview: true,
		enabled:        token != "" && chatID != 0,
	}
	for _, opt := range opts {
		opt(&st)
	}
	tg.Store(st)
}

func TelegramEnabled() bool {
	st, _ := tg.Load().(telegramState)
	return st.enabled
}

type TelegramOption func(*telegramState)

func WithTelegramHTTPClient(c *http.Client) TelegramOption {
	return func(s *telegramState) {
		if c != nil {
			s.httpClient = c
		}
	}
}

func WithTelegramParseMode(mode string) TelegramOption {
	return func(s *telegramState) {
		s.parseMode = mode
	}
}

func WithTelegramDisablePreview(disable bool) TelegramOption {
	return func(s *telegramState) {
		s.disablePreview = disable
	}
}

func TelegramSend(ctx context.Context, text string) error {
	st, _ := tg.Load().(telegramState)
	if !st.enabled || text == "" {
		return nil
	}
	if _, has := ctx.Deadline(); !has {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	for _, chunk := range splitByLen(text, tgMaxMsgLen) {
		if err := telegramSendOne(ctx, st, chunk); err != nil {
			return err
		}
	}
	return nil
}

func TelegramSendAsync(text string) {
	go func() {
		_ = TelegramSend(context.Background(), text)
	}()
}

// с ретраями (2 попытки), обработкой 429/5xx
func telegramSendOne(ctx context.Context, st telegramState, text string) error {
	if !st.enabled {
		return nil
	}
	reqBody := map[string]any{
		"chat_id":                  st.chatID,
		"text":                     text,
		"disable_web_page_preview": st.disablePreview,
	}
	if st.parseMode != "" {
		reqBody["parse_mode"] = st.parseMode
	}
	data, _ := json.Marshal(reqBody)
	url := "https://api.telegram.org/bot" + st.token + "/sendMessage"

	var lastErr error
	const attempts = 2

	for attempt := 1; attempt <= attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := st.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				var res struct {
					OK          bool   `json:"ok"`
					Description string `json:"description"`
				}
				if err := json.Unmarshal(bodyBytes, &res); err == nil {
					if res.OK {
						return nil
					}
					if res.Description != "" {
						return errors.New(res.Description)
					}
					return errors.New("telegram: response not ok")
				}
				return fmt.Errorf("telegram: malformed success response")
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				// 429 — у Телеграма retry_after
				var wait time.Duration
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if sec, e := strconv.Atoi(ra); e == nil && sec > 0 {
						wait = time.Duration(sec) * time.Second
					}
				}
				if wait == 0 {
					var te struct {
						Parameters struct {
							RetryAfter int `json:"retry_after"`
						} `json:"parameters"`
						Description string `json:"description"`
					}
					_ = json.Unmarshal(bodyBytes, &te)
					if te.Parameters.RetryAfter > 0 {
						wait = time.Duration(te.Parameters.RetryAfter) * time.Second
					}
					if te.Description != "" {
						lastErr = errors.New(te.Description)
					} else {
						lastErr = fmt.Errorf("telegram 429")
					}
				}
				if wait == 0 {
					wait = 2 * time.Second
				}
				if attempt < attempts {
					t := time.NewTimer(wait)
					select {
					case <-ctx.Done():
						t.Stop()
						return ctx.Err()
					case <-t.C:
					}
					continue
				}
				return lastErr
			}

			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				lastErr = fmt.Errorf("telegram http status: %d", resp.StatusCode)
				if attempt < attempts {
					backoff := time.Duration(attempt) * 500 * time.Millisecond
					t := time.NewTimer(backoff)
					select {
					case <-ctx.Done():
						t.Stop()
						return ctx.Err()
					case <-t.C:
					}
					continue
				}
				return lastErr
			}

			var te struct {
				Description string `json:"description"`
			}
			if err := json.Unmarshal(bodyBytes, &te); err == nil && te.Description != "" {
				return errors.New(te.Description)
			}
			return fmt.Errorf("telegram http status: %d", resp.StatusCode)
		}

		if attempt < attempts {
			t := time.NewTimer(500 * time.Millisecond)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			continue
		}
	}
	return lastErr
}

func splitByLen(s string, max int) []string {
	runes := []rune(s)
	if len(runes) <= max {
		return []string{s}
	}
	res := make([]string, 0, (len(runes)/max)+1)
	for i := 0; i < len(runes); i += max {
		j := i + max
		if j > len(runes) {
			j = len(runes)
		}
		res = append(res, string(runes[i:j]))
	}
	return res
}
