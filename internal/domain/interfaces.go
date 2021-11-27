package domain

import (
	"context"
	"lesson1/internal/models"
)

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth uint64)
	ChanResult() <-chan models.CrawlResult
	AddDepth(delta uint64)
}
