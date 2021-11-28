package crawler

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"lesson1/internal/domain"
	"testing"
	"time"
)

var (
	startUrl = "https://golang.org"
)

func TestCrawler_Scan(t *testing.T) {
	wantLinks := []string{
		"url: https://golang.org, title: Home",
		"url: https://golang1.org, title: Home",
		"url: https://golang2.org, title: Home",
		"url: https://golang3.org, title: Home",
	}
	//r := mocks.Requester{}
	r := mockRequester{}
	crawler, _ := NewCrawler(&r, 3, nil)
	ctx := context.Background()

	go crawler.Scan(ctx, startUrl, 1)
	var res []string
	var next = true
	for next {
		select {
		case <-time.After(time.Duration(5) * time.Second):
			next = false
		case msg := <-crawler.ChanResult():
			fmt.Println(msg)
			res = append(res, fmt.Sprintf("url: %s, title: %s", msg.Url, msg.Title))
		}
	}
	assert.ElementsMatch(t, wantLinks, res)
	t.Log("success")
}

type mockPage struct{}

func (mp *mockPage) GetTitle() string {
	return "Home"
}

func (mp *mockPage) GetLinks() []string {
	return []string{
		"https://golang1.org",
		"https://golang2.org",
		"https://golang3.org",
	}
}

type mockRequester struct{}

func (m *mockRequester) Get(ctx context.Context, url string) (domain.Page, error) {
	var mp mockPage
	return &mp, nil
}
