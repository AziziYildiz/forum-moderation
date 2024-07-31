package handlers

import (
	"fmt"
	"forum/utils"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var funcMap = template.FuncMap{
	"ToLower": strings.ToLower,
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	var tmpl *template.Template
	var err error

	users, err := getAllUsers()
	if err != nil {
		log.Println("Error fetching users:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	requests, err := getModerationRequests()
	if err != nil {
		log.Println("Error fetching moderation requests:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	posts, err := getAllPosts()
	if err != nil {
		log.Println("Error fetching posts:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	reports, err := getAllPostReports()
	if err != nil {
		log.Println("Error fetching post reports:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	comments, err := getAllComments()
	if err != nil {
		log.Println("Error fetching comments:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pageData := AdminPageData{
		Users:              users,
		ModerationRequests: requests,
		Posts:              posts,
		Reports:            reports,
		Comments:           comments,
	}

	switch view {
	case "", "users":
		tmpl, err = template.New("admin_users.html").Funcs(funcMap).ParseFiles("templates/admin_users.html")
		if err != nil {
			log.Println("Error loading admin_users.html template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "posts":
		tmpl, err = template.New("admin_posts.html").Funcs(funcMap).ParseFiles("templates/admin_posts.html")
		if err != nil {
			log.Println("Error loading admin_posts.html template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "moderation_requests":
		tmpl, err = template.New("moderation_requests.html").Funcs(funcMap).ParseFiles("templates/moderation_requests.html")
		if err != nil {
			log.Println("Error loading moderation_requests.html template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "post_reports":
		tmpl, err = template.New("post_reports.html").Funcs(funcMap).ParseFiles("templates/post_reports.html")
		if err != nil {
			log.Println("Error loading post_reports.html template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	case "comments":
		tmpl, err = template.New("admin_comments.html").Funcs(funcMap).ParseFiles("templates/admin_comments.html")
		if err != nil {
			log.Println("Error loading admin_comments.html template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	default:
		http.NotFound(w, r)
		return
	}

	err = tmpl.Execute(w, pageData)
	if err != nil {
		log.Println("Error executing template:", err)
	}
}

func getAllComments() ([]Comment, error) {
	rows, err := utils.Db.Query(`
        SELECT c.id, c.content, u.username, 
        COALESCE(SUM(CASE cv.vote WHEN 1 THEN 1 ELSE 0 END), 0) AS likes,
        COALESCE(SUM(CASE cv.vote WHEN -1 THEN 1 ELSE 0 END), 0) AS dislikes
        FROM comments c
        JOIN users u ON c.author_id = u.id
        LEFT JOIN comment_votes cv ON c.id = cv.comment_id
        GROUP BY c.id, u.username
        ORDER BY c.created_at DESC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.ID, &comment.Content, &comment.AuthorName, &comment.Likes, &comment.Dislikes); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func adminDeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Admin check
	isAdmin := utils.IsAdmin(r)
	if !isAdmin {
		http.Error(w, "Unauthorized to delete this comment", http.StatusForbidden)
		return
	}

	commentID, err := strconv.Atoi(r.FormValue("comment_id"))
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	// Delete the comment
	_, err = utils.Db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		http.Error(w, "Error deleting comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin?view=comments", http.StatusSeeOther)
}

func submitReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := utils.GetUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Kullanıcı adını veritabanından al
	var username string
	err = utils.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	postIDStr := r.FormValue("post_id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	reportContent := r.FormValue("report_content")
	if reportContent == "" {
		http.Error(w, "Report content cannot be empty", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	// Post tablosundaki postreport sütununa rapor mesajını ekle
	report := fmt.Sprintf("UserID: %d, Username: %s, Message: %s, Report: %s", userID, username, message, reportContent)
	_, err = utils.Db.Exec("UPDATE posts SET postreport = ? WHERE id = ?", report, postID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/view_post?post_id=%d", postID), http.StatusSeeOther)
}

func deletePostReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postIDStr := r.FormValue("post_id")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	_, err = utils.Db.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin?view=post_reports", http.StatusSeeOther)
}

func getAllPostReports() ([]PostReport, error) {
	query := `
		SELECT p.id, p.author_id, u.username, p.postreport, p.created_at
		FROM posts p
		JOIN users u ON p.author_id = u.id
		WHERE p.postreport IS NOT NULL
	`
	rows, err := utils.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []PostReport
	for rows.Next() {
		var report PostReport
		var postReport string
		if err := rows.Scan(&report.ID, &report.UserID, &report.Username, &postReport, &report.CreatedAt); err != nil {
			return nil, err
		}

		// Extract only the message part from the report
		parts := strings.Split(postReport, ", ")
		for _, part := range parts {
			if strings.HasPrefix(part, "Report: ") {
				report.Content = strings.TrimPrefix(part, "Report: ")
			}
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func getAllUsers() ([]User, error) {
	rows, err := utils.Db.Query("SELECT id, username, name, about, usericon_url, email, role FROM users WHERE role != 'admin'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.Name, &user.About, &user.UserIconURL, &user.Email, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func getAllPosts() ([]Post, error) {
	query := `
        SELECT 
            p.id, p.title, p.content, p.image_url, 
            p.author_id, u.username AS author_name,
            p.created_at, 
            COALESCE((SELECT GROUP_CONCAT(pc.category) FROM post_categories pc WHERE pc.post_id = p.id), '') AS category,
            (SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count
        FROM posts p
        JOIN users u ON p.author_id = u.id
    `

	rows, err := utils.Db.Query(query)
	if err != nil {
		log.Printf("Error querying posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.ImageURL, &post.AuthorID, &post.AuthorName, &post.CreatedAt, &post.Category, &post.CommentCount)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			return nil, err
		}

		// HTML etiketlerini içerikten temizle

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error reading posts: %v", err)
		return nil, err
	}

	return posts, nil
}

func getModerationRequests() ([]ModerationRequest, error) {
	rows, err := utils.Db.Query("SELECT id, user_id, username, message, created_at FROM moderation_requests")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []ModerationRequest
	for rows.Next() {
		var request ModerationRequest
		if err := rows.Scan(&request.ID, &request.UserID, &request.Username, &request.Message, &request.CreatedAt); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func ChangeUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.FormValue("userId")
	newRole := r.FormValue("newRole")

	// Kullanıcı bilgilerini kontrol et
	user, err := getUserByID(userID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if user.Role == "Admin" {
		http.Error(w, "Cannot demote admin role", http.StatusBadRequest)
		return
	}

	// Role güncelleme
	if _, err := utils.Db.Exec("UPDATE users SET role = ? WHERE id = ?", newRole, userID); err != nil {
		log.Println("Error updating role:", err) // Hata detayını logla
		http.Error(w, "Failed to update role", http.StatusInternalServerError)
		return
	}

	// Aynı sayfaya yönlendir
	http.Redirect(w, r, "/moderation-requests", http.StatusSeeOther)
}

func getUserByID(userID string) (User, error) {
	var user User
	err := utils.Db.QueryRow("SELECT id, username, email, role FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err != nil {
		return user, err
	}
	return user, nil
}

func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")
	var deleteQuery string
	if action == "delete_user" {
		deleteQuery = "DELETE FROM users WHERE id = ?"
	} else if action == "delete_post" {
		deleteQuery = "DELETE FROM posts WHERE id = ?"
	} else {
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	_, err = utils.Db.Exec(deleteQuery, id)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	redirectURL := "/admin"
	if action == "delete_user" {
		redirectURL += "?view=users"
	} else if action == "delete_post" {
		redirectURL += "?view=posts"
	}

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func applyModeratorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID, err := utils.GetUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	// Kullanıcı adını veritabanından almak
	var username string
	err = utils.Db.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)
	if err != nil {
		http.Error(w, "Error fetching username", http.StatusInternalServerError)
		return
	}

	createdAt := time.Now()

	_, err = utils.Db.Exec("INSERT INTO moderation_requests (user_id, username, message, created_at) VALUES (?, ?, ?, ?)", userID, username, message, createdAt)
	if err != nil {
		http.Error(w, "Error submitting moderation request", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func mod_deletePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userID, err := utils.GetUserIDFromCookie(r)
	if err != nil || !utils.CheckLoginStatus(r) {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Check if the user is a moderator
	isModerator, err := utils.IsModerator(userID, utils.Db)
	if err != nil {
		http.Error(w, "Error checking moderator status", http.StatusInternalServerError)
		return
	}

	if !isModerator {
		http.Error(w, "Unauthorized to delete this post", http.StatusForbidden)
		return
	}

	// Post'u sil
	res, err := utils.Db.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Postun gerçekten silindiğinden emin olun
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, "Error checking delete result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "No post found with the given ID", http.StatusNotFound)
		return
	}

	// Post silindikten sonra ana sayfaya yönlendir
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func pending(w http.ResponseWriter, r *http.Request) {
	// Get user ID from session
	userID, err := utils.GetUserIDFromCookie(r)
	if err != nil {
		log.Printf("Error getting user ID: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Check if the user is a moderator
	isModerator, err := utils.IsModerator(userID, utils.Db)
	if err != nil {
		log.Printf("Error checking if user is moderator: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !isModerator {
		http.Redirect(w, r, "/", http.StatusForbidden)
		return
	}

	// Fetch posts by status
	pendingPosts, err := fetchPostsByStatus("pending")
	if err != nil {
		log.Printf("Error fetching pending posts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	approvedPosts, err := fetchPostsByStatus("approved")
	if err != nil {
		log.Printf("Error fetching approved posts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rejectedPosts, err := fetchPostsByStatus("rejected")
	if err != nil {
		log.Printf("Error fetching rejected posts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Load and execute the template with posts
	tmpl, err := template.ParseFiles("templates/moderator_page.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	data := struct {
		PendingPosts  []Post
		ApprovedPosts []Post
		RejectedPosts []Post
	}{
		PendingPosts:  pendingPosts,
		ApprovedPosts: approvedPosts,
		RejectedPosts: rejectedPosts,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}

func fetchPostsByStatus(status string) ([]Post, error) {
	var posts []Post
	query := `
        SELECT p.id, p.author_id, p.title, p.image_url, u.username, p.content, p.created_at,
        COALESCE((SELECT GROUP_CONCAT(pc.category) FROM post_categories pc WHERE pc.post_id = p.id), '') AS category
        FROM posts p
        JOIN users u ON p.author_id = u.id
        WHERE p.status = ?
        ORDER BY p.created_at DESC
    `

	rows, err := utils.Db.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.AuthorID, &p.Title, &p.ImageURL, &p.AuthorName, &p.Content, &p.CreatedAt, &p.Category); err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func approvePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("id")
	_, err := utils.Db.Exec("UPDATE posts SET status = 'approved' WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// İsteği aldıktan sonra, moderator sayfasına yönlendir.
	http.Redirect(w, r, "/pending", http.StatusSeeOther)
}

func rejectPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		return
	}

	postID := r.FormValue("id")
	_, err := utils.Db.Exec("UPDATE posts SET status = 'rejected' WHERE id = ?", postID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// İsteği aldıktan sonra, moderator sayfasına yönlendir.
	http.Redirect(w, r, "/pending", http.StatusSeeOther)
}
