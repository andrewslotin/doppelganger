package github

import "time"

type Commit struct {
	SHA     string
	Message string
	Author  string
	Date    time.Time
}
