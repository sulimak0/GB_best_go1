package page

import (
	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	"io"
)

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
