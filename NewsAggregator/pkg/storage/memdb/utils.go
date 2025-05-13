package memdb

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/gofrs/uuid"

	"news/pkg/storage"
)

// LoadTestPosts loads test posts from a JSON file at the given path, converting
// their published dates to UTC and sorting them in descending order by date.
func LoadTestPosts(path string) ([]storage.Post, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to load test data from file: %w", err)
	}

	var posts []storage.Post
	err = json.Unmarshal(b, &posts)

	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal test data: %w", err)
	}

	for i := 0; i < len(posts); i++ {
		posts[i].ID = uuid.NewV5(uuid.NamespaceURL, posts[i].Link)
		posts[i].Published = posts[i].Published.UTC()
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Published.After(posts[j].Published)
	})

	return posts, nil
}
