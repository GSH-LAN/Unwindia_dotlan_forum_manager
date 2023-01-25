package dotlan

import (
	"time"
)

type ForumThread struct {
	Threadid     uint      `db:"threadid"`
	Title        string    `db:"title"`
	Forumid      int       `db:"forumid"`
	Closed       bool      `db:"closed"`
	Invisible    bool      `db:"invisible"`
	ShowLatest   bool      `db:"show_latest"`
	sticky       bool      `db:"sticky"`
	User_id      int       `db:"user_id"`
	Firstposter  string    `db:"firstposter"`
	Lastposter   string    `db:"lastposter"`
	Lastposttime time.Time `db:"lastposttime"`
	Replies      int       `db:"replies"`
	Hits         int       `db:"hits"`
	Fv_id        int       `db:"fv_id"`
	Ext          string    `db:"ext"`
	ExtId        uint      `db:"ext_id"`
}

func (ForumThread) TableName() string {
	return "forum_thread"
}

type ForumPost struct {
	Postid       uint      `db:"postid"`
	Threadid     uint      `db:"threadid"`
	Userid       int       `db:"userid"`
	Dateline     time.Time `db:"dateline"`
	Pagetext     string    `db:"pagetext"`
	Htmltext     string    `db:"htmltext"`
	Remoteip     string    `db:"remoteip"`
	Dnsname      string    `db:"dnsname"`
	Locked       bool      `db:"locked"`
	Invisible    bool      `db:"invisible"`
	History      string    `db:"history"`
	Attachment_i int       `db:"attachment_i"`
	Notify       int       `db:"notify"`
	Quoteid      int       `db:"quoteid"`
}

func (ForumPost) TableName() string {
	return "forum_post"
}
