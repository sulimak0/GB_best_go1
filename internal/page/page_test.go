package page

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var (
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

func TestPage_GetTitle(t *testing.T) {
	tPage, _ := NewPage(strings.NewReader(testPage), nil)
	links := tPage.GetLinks()
	want := []string{
		"https://golang1.org",
		"https://golang2.org",
		"https://golang3.org",
	}
	assert.ElementsMatch(t, want, links)
	t.Logf("correct")
}
