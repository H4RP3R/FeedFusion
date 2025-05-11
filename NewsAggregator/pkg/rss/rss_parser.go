package rss

import (
	"encoding/json"
	"os"
	"time"

	"github.com/mmcdole/gofeed"

	"news/pkg/storage"
)

type ParserMsg struct {
	Source string
	Data   []storage.Post
	Err    error
}

type config struct {
	RSS           []string `json:"rss"`
	RequestPeriod int      `json:"request_period"`
}

func LoadConf(path string) (*config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf config
	err = json.Unmarshal(b, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
}

type Parser struct {
	Urls  []string
	Delay time.Duration
}

// handleFeed processes a single RSS feed, converting its items to storage.Post
// objects and returning them as a slice. Converts the post's Published field to UTC.
func (p *Parser) handleFeed(feed *gofeed.Feed) ([]storage.Post, error) {
	var posts []storage.Post

	for _, item := range feed.Items {
		publishedUTC, _ := ConvertToUTC(item.Published)
		post := storage.Post{
			Title:     item.Title,
			Content:   item.Description,
			Published: publishedUTC,
			Link:      item.Link,
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// Run runs the parser for each URL in a separate goroutine.
func (p *Parser) Run(msgChan chan<- ParserMsg) {
	for _, url := range p.Urls {
		go func(url string) {
			msg := ParserMsg{Source: url}
			fp := gofeed.NewParser()

			feed, err := fp.ParseURL(url)
			if err != nil {
				msg.Err = err
				msgChan <- msg
				return
			}

			posts, err := p.handleFeed(feed)
			if err != nil {
				msg.Err = err
				msgChan <- msg
				return
			}

			msg.Data = posts
			msgChan <- msg

		}(url)
	}
}

func NewParser(conf config) *Parser {
	p := Parser{
		Urls:  conf.RSS,
		Delay: time.Minute * time.Duration(conf.RequestPeriod),
	}

	return &p
}
