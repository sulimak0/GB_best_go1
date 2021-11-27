package models

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}
