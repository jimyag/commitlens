package git

import "time"

type Commit struct {
	SHA          string    `json:"sha"`
	Author       string    `json:"author"`
	AuthorEmail  string    `json:"author_email"`
	Participants []string  `json:"participants"`
	Message      string    `json:"message"`
	Date         time.Time `json:"date"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
}

// LineCountPoint is the total number of lines in tracked text files after a
// commit on the repository's default branch.
type LineCountPoint struct {
	SHA   string
	Date  time.Time
	Lines int
}
