package main

import (
	"context"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type page struct {
	doc  *goquery.Document
	slog *zap.SugaredLogger
}

func NewPage(raw io.Reader, slog *zap.SugaredLogger) (*page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		slog.Debugf("can't be parsed: %s", err)
		return nil, err
	}
	return &page{doc: doc, slog: slog}, nil
}

func (p *page) GetTitle() string {
	return p.doc.Find("title").First().Text()
}

func (p *page) GetLinks() []string {
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			urls = append(urls, url)
		}
	})
	return urls
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout time.Duration
	slog    *zap.SugaredLogger
}

func NewRequester(timeout time.Duration, slog *zap.SugaredLogger) requester {
	return requester{timeout: timeout, slog: slog}
}

func (r requester) Get(ctx context.Context, url string) (Page, error) {
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
		page, err := NewPage(body.Body, r.slog)
		if err != nil {
			r.slog.Debugf("can't create page: %s", err)
			return nil, err
		}
		return page, nil
	}
	return nil, nil
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
	AddDepth(delta int)
}

type crawler struct {
	r        Requester
	res      chan CrawlResult
	visited  map[string]struct{}
	mu       sync.RWMutex
	maxDepth int
	slog     *zap.SugaredLogger
}

func NewCrawler(r Requester, maxDepth int, slog *zap.SugaredLogger) *crawler {
	return &crawler{
		r:        r,
		res:      make(chan CrawlResult),
		visited:  make(map[string]struct{}),
		mu:       sync.RWMutex{},
		maxDepth: maxDepth,
		slog:     slog,
	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth int) {
	if depth >= c.maxDepth+1 { //Проверяем то, что есть запас по глубине
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
	if err != nil {
		c.slog.Debugf("can't get page: %s", err)
		c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
		return
	}
	c.mu.Lock()
	c.visited[url] = struct{}{} //Помечаем страницу просмотренной
	c.mu.Unlock()
	c.res <- CrawlResult{ //Отправляем результаты в канал
		Title: page.GetTitle(),
		Url:   url,
	}
	for {
		select {
		case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
			return
		default:
			if c.maxDepth > depth {
				for _, link := range page.GetLinks() {
					newDepth := depth + 1
					c.slog.Debugw("started new Scan goroutine", "depth", newDepth)
					go c.Scan(ctx, link, newDepth) //На все полученные ссылки запускаем новую рутину сборки
				}
				return
			}
		}
	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}

func (c *crawler) AddDepth(delta int) {
	c.mu.Lock()
	c.maxDepth += delta
	c.mu.Unlock()
}

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't init logger: %s", err)
	}
	defer logger.Sync()
	slog := logger.Sugar()
	slog.Infof("init logger")

	cfg := Config{
		MaxDepth:   1,
		MaxResults: 10000,
		MaxErrors:  10000,
		Url:        "https://golang.org",
		Timeout:    15,
	}
	slog.Infow("read config", "config", cfg)
	slog.Debugw("process id", "id", os.Getpid())

	var cr Crawler
	var r Requester

	r = NewRequester(time.Duration(cfg.Timeout)*time.Second, slog)
	cr = NewCrawler(r, cfg.MaxDepth, slog)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	//ctx, cancel := context.WithCancel(context.Background())

	go cr.Scan(ctx, cfg.Url, 0)                  //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg, slog) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal)                          //Создаем канал для приема сигналов
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT
	for {
		select {
		case <-ctx.Done():
			slog.Warnw("ctx done")
			return
		case s := <-sigCh:
			switch s {
			case syscall.SIGUSR1:
				slog.Infow("SIGURSR1")
				cr.AddDepth(2)
			case syscall.SIGTERM:
				slog.Infow("SIGTERM")
				cancel()
			}
		}
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config, slog *zap.SugaredLogger) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				slog.Warnf("crawler result return err: %s", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					return
				}
			} else {
				maxResult--
				slog.Infof("crawler result: [url: %s] Title: %s", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
