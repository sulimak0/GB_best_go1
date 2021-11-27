package requester

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"lesson1/internal/domain"
	"lesson1/internal/page"
	"net/http"
	"time"
)

var (
	errIncorrectTimeout = errors.New("incorrect timeout value, should be > 0")
)

type requester struct {
	timeout time.Duration
	slog    *zap.SugaredLogger
}

func NewRequester(timeout time.Duration, slog *zap.SugaredLogger) (*requester, error) {
	if timeout <= 0 {
		return nil, errIncorrectTimeout
	}
	return &requester{timeout: timeout, slog: slog}, nil
}

func (r requester) Get(ctx context.Context, url string) (domain.Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			r.slog.Debugf("can't create http.Request: %s", err)
			return nil, err
		}
		body, err := cl.Do(req)
		if err != nil {
			r.slog.Debugf("http transport error: %s", err)
			return nil, err
		}
		defer body.Body.Close()
		page, err := page.NewPage(body.Body, r.slog)
		if err != nil {
			r.slog.Debugf("can't create page: %s", err)
			return nil, err
		}
		return page, nil
	}
	return nil, nil
}
