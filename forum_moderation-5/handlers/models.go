package handlers

import (
	"database/sql"
	"time"
)

type ModeratorRequest struct {
	ID       int
	Username string
	Request  string
}

type Post struct {
	ID                 int
	Title              string
	ImageURL           string
	Content            string
	AuthorName         string
	AuthorIcon         sql.NullString
	Likes              int
	Dislikes           int
	UserIcon           string
	Category           sql.NullString
	CreatedAt          time.Time
	FormattedCreatedAt string
	CommentCount       int
	ViewCount          int
	AuthorID           int
	IsModerator        bool
	Status             string
}

type User struct {
	ID          int
	Email       string
	Username    string
	Name        sql.NullString
	About       sql.NullString
	UserIconURL sql.NullString
	CreatedAt   time.Time
	Role        string
	Password    string
	ErrorMsg    string
}

type ModerationRequest struct {
	ID        int
	UserID    int
	Username  string
	Message   string
	CreatedAt time.Time
	Role      string
}

type PostReport struct {
	ID        int
	UserID    int
	Username  string
	Content   string
	CreatedAt time.Time
}

type AdminPageData struct {
	Users              []User
	ModerationRequests []ModerationRequest
	Posts              []Post
	Reports            []PostReport
	Comments           []Comment
}

