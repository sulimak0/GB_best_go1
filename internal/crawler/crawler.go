package crawler

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"lesson1/internal/domain"
	"lesson1/internal/models"
	"sync"
	"sync/atomic"
)

var (
	errIncorrectMaxDepth = errors.New("incorrect maxDepth")
)

type crawler struct {
	r        domain.Requester
	res      chan models.CrawlResult
	visited  map[string]struct{}
	mu       sync.RWMutex
	maxDepth uint64
	slog     *zap.SugaredLogger
}

func NewCrawler(r domain.Requester, maxDepth uint64, slog *zap.SugaredLogger) (*crawler, error) {
	if maxDepth < 1 {
		return nil, errIncorrectMaxDepth
	}

	return &crawler{
		r:        r,
		res:      make(chan models.CrawlResult),
		visited:  make(map[string]struct{}),
		mu:       sync.RWMutex{},
		maxDepth: maxDepth,
		slog:     slog,
	}, nil
}

func (c *crawler) Scan(ctx context.Context, url string, depth uint64) {
	//Проверяем то, что есть запас по глубине
	c.mu.RLock()
	newDepth := depth > c.maxDepth
	c.mu.RUnlock()
	if newDepth {
		return
	}

	//Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RLock()
	_, ok := c.visited[url]
	c.mu.RUnlock()
	if ok {
		return
	}
	select {
	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		return
	default:
		//Запрашиваем страницу через Requester
		p, err := c.r.Get(ctx, url)
		if err != nil {
			c.slog.Debugf("can't get page: %s", err)
			c.res <- models.CrawlResult{Err: err}
			return
		}
		//Помечаем страницу просмотренной и отправляем резуальтат в канал
		c.mu.Lock()
		c.visited[url] = struct{}{}
		c.mu.Unlock()
		c.res <- models.CrawlResult{
			Title: p.GetTitle(),
			Url:   url,
		}
		for _, link := range p.GetLinks() {
			go c.Scan(ctx, link, depth+1) //На все полученные ссылки запускаем новую рутину сборки
		}

	}

}

func (c *crawler) ChanResult() <-chan models.CrawlResult {
	return c.res
}

func (c *crawler) AddDepth(delta uint64) {
	newDepth := atomic.AddUint64(&c.maxDepth, delta)
	c.slog.Debugw("increase delta value", "depth", newDepth)
}
