package requester

import (
	"context"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var (
	startUrl = "https://golang.org"
	testPage = `<!doctype html>
		<html lang="en">
			<head>
    			<meta charset="utf-8">
    			<title>Home</title>
			</head>
			<body>
    			<h1>Mock</h1>
    			<a href="https://golang1.org">Golang1</a>
				<a href="https://golang2.org">Golang1</a>
				<a href="https://golang3.org">Golang1</a>
			</body>
		</html>`
)

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (s roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

func TestRequester_Get(t *testing.T) {
	requester, _ := NewRequester(1, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(testPage)),
		}, nil
	}), nil)
	links, _ := requester.Get(context.Background(), startUrl)

	wantTitle := "Home"
	if links.GetTitle() != wantTitle {
		t.Errorf("title are not equal. want: %s, got: %s", wantTitle, links.GetTitle())
	}

	wantLinks := []string{
		"https://golang1.org",
		"https://golang2.org",
		"https://golang3.org",
	}

	assert.ElementsMatch(t, wantLinks, links.GetLinks())
	t.Logf("correct")
}
