package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	db  *sql.DB
	cld *cloudinary.Cloudinary
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
	Role      string `json:"role"`
	IsBanned  bool   `json:"is_banned"`
	Bio       string `json:"bio"`
	AvatarURL string `json:"avatar_url"`
}

type Reaction struct {
	Emoji       string `json:"emoji"`
	Count       int    `json:"count"`
	UserReacted bool   `json:"user_reacted"`
}

type ReactionUser struct {
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Emoji     string `json:"emoji"`
}

type Comment struct {
	ID         int        `json:"id"`
	PostID     int        `json:"post_id"`
	UserID     int        `json:"user_id"`
	ParentID   *int       `json:"parent_id,omitempty"`
	AuthorName string     `json:"author_name"`
	AuthorRole string     `json:"author_role"`
	AuthorPic  string     `json:"author_pic"`
	Content    string     `json:"content"`
	CreatedAt  string     `json:"created_at"`
	Reactions  []Reaction `json:"reactions"`
	Replies    []Comment  `json:"replies"`
}

type Post struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	AuthorName  string     `json:"author_name"`
	AuthorRole  string     `json:"author_role"`
	AuthorPic   string     `json:"author_pic"`
	Content     string     `json:"content"`
	ImageURL    string     `json:"image_url"`
	IsAnonymous bool       `json:"is_anonymous"`
	Status      string     `json:"status"`
	CreatedAt   string     `json:"created_at"`
	Reactions   []Reaction `json:"reactions"`
	Comments    []Comment  `json:"comments"`
}

type Notification struct {
	ID        int    `json:"id"`
	ActorName string `json:"actor_name"`
	ActorPic  string `json:"actor_pic"`
	Type      string `json:"type"`
	PostID    int    `json:"post_id"`
	Message   string `json:"message"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	URL     string `json:"url,omitempty"`
	User    *User  `json:"user,omitempty"`
}

func initDB() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Println("⚠️ DATABASE_URL not set!")
		return
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v\n", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to connect to Neon DB: %v\n", err)
	}

	ensureAdminAccount()
	fmt.Println("✅ Connected to Neon PostgreSQL database!")
}

func ensureAdminAccount() {
	// 1. Ensure original Admin (gabby) exists
	var gabbyID int
	err := db.QueryRow(`SELECT id FROM users WHERE username = 'gabby'`).Scan(&gabbyID)
	if err == sql.ErrNoRows {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("gabbytheowner"), bcrypt.DefaultCost)
		_, err := db.Exec(`INSERT INTO users (username, nickname, email, password, role) VALUES ('gabby', 'Gabby (Owner)', 'gabby@corstellae.com', $1, 'admin')`, string(hashed))
		if err == nil {
			fmt.Println("👑 Hardcoded Admin Account Initialized: gabby / gabbytheowner")
		}
	} else {
		db.Exec(`UPDATE users SET role = 'admin' WHERE username = 'gabby'`)
	}

	// 2. Ensure Developer Account (haroldharvey) exists with developer role
	var devID int
	err = db.QueryRow(`SELECT id FROM users WHERE username = 'haroldharvey'`).Scan(&devID)
	if err == sql.ErrNoRows {
		hashedDev, _ := bcrypt.GenerateFromPassword([]byte("harold123123"), bcrypt.DefaultCost)
		_, err := db.Exec(`INSERT INTO users (username, nickname, email, password, role) VALUES ('haroldharvey', 'Harold Harvey', 'harold@corstellae.com', $1, 'developer')`, string(hashedDev))
		if err == nil {
			fmt.Println("💻 Hardcoded Developer Account Initialized: haroldharvey / harold123123")
		}
	} else {
		db.Exec(`UPDATE users SET role = 'developer' WHERE username = 'haroldharvey'`)
	}
}

func initCloudinary() {
	cldURL := os.Getenv("CLOUDINARY_URL")
	if cldURL == "" {
		log.Println("⚠️ CLOUDINARY_URL not set!")
		return
	}

	var err error
	cld, err = cloudinary.NewFromURL(cldURL)
	if err != nil {
		log.Fatalf("Failed to init Cloudinary: %v\n", err)
	}
	fmt.Println("✅ Connected to Cloudinary!")
}

func main() {
	initDB()
	initCloudinary()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Health Endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Page Routes
	http.HandleFunc("/", serveLogin)
	http.HandleFunc("/dashboard", serveDashboard)

	// Auth & Profile APIs
	http.HandleFunc("/api/register", handleRegister)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/logout", handleLogout)
	http.HandleFunc("/api/user/profile", handleUserProfile)
	http.HandleFunc("/api/user/update", handleUpdateProfile)

	// Post APIs
	http.HandleFunc("/api/posts/create", handleCreatePost)
	http.HandleFunc("/api/posts/public", handleGetPublicPosts)
	http.HandleFunc("/api/posts/edit", handleEditPost)
	http.HandleFunc("/api/posts/delete", handleDeletePost)

	// Reaction & Comment APIs
	http.HandleFunc("/api/reactions/toggle", handleToggleReaction)
	http.HandleFunc("/api/reactions/users", handleGetReactionUsers)
	http.HandleFunc("/api/comments/create", handleCreateComment)
	http.HandleFunc("/api/comments/delete", handleDeleteComment)

	// Notification APIs
	http.HandleFunc("/api/notifications", handleGetNotifications)
	http.HandleFunc("/api/notifications/read", handleMarkNotificationsRead)

	// Admin APIs
	http.HandleFunc("/api/admin/users", handleGetUsersList)
	http.HandleFunc("/api/admin/user/role", handleUpdateUserRole)
	http.HandleFunc("/api/admin/user/ban", handleBanUser)
	http.HandleFunc("/api/admin/pending", handleGetPendingPosts)
	http.HandleFunc("/api/admin/review", handleReviewPost)
	// Admin Comment Moderation APIs
	http.HandleFunc("/api/admin/comments/pending", handleGetPendingComments)
	http.HandleFunc("/api/admin/comments/review", handleReviewComment)

	// Media Upload API
	http.HandleFunc("/api/upload", handleUpload)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("CorStellae Go server running on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if cookie, err := r.Cookie("session_user_id"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "./templates/login.html")
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "./templates/index.html")
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u User
	json.NewDecoder(r.Body).Decode(&u)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)

	query := `INSERT INTO users (username, nickname, email, password, role) VALUES ($1, $2, $3, $4, 'user') RETURNING id`
	err := db.QueryRow(query, u.Username, u.Nickname, u.Email, string(hashedPassword)).Scan(&u.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Status: "error", Message: "Username or Email already exists."})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Account created! You can now log in."})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		EmailOrUser string `json:"email_or_user"`
		Password    string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var u User
	var storedHash string
	query := `SELECT id, username, nickname, email, password, role, is_banned, COALESCE(bio, ''), COALESCE(avatar_url, '') 
	          FROM users WHERE email = $1 OR username = $1`
	err := db.QueryRow(query, input.EmailOrUser).Scan(&u.ID, &u.Username, &u.Nickname, &u.Email, &storedHash, &u.Role, &u.IsBanned, &u.Bio, &u.AvatarURL)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(input.Password)) != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{Status: "error", Message: "Invalid username or password."})
		return
	}

	if u.IsBanned {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Status: "error", Message: "This account has been banned."})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_user_id",
		Value:    strconv.Itoa(u.ID),
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	u.Password = ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Login successful!", User: &u})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_user_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Logged out"})
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	var u User
	query := `SELECT id, username, nickname, email, role, COALESCE(bio, ''), COALESCE(avatar_url, '') FROM users WHERE id = $1`
	err := db.QueryRow(query, userID).Scan(&u.ID, &u.Username, &u.Nickname, &u.Email, &u.Role, &u.Bio, &u.AvatarURL)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID    int    `json:"user_id"`
		Nickname  string `json:"nickname"`
		Bio       string `json:"bio"`
		AvatarURL string `json:"avatar_url"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	_, err := db.Exec(`UPDATE users SET nickname = $1, bio = $2, avatar_url = $3 WHERE id = $4`, input.Nickname, input.Bio, input.AvatarURL, input.UserID)
	if err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Profile updated!"})
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID      int    `json:"user_id"`
		Content     string `json:"content"`
		ImageURL    string `json:"image_url"`
		IsAnonymous bool   `json:"is_anonymous"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = $1`, input.UserID).Scan(&role)

	status := "pending"
	// Developer, Admin, and Moderator posts bypass moderation and are auto-approved
	if role == "developer" || role == "admin" || role == "moderator" {
		status = "approved"
	}

	query := `INSERT INTO posts (user_id, content, image_url, is_anonymous, status) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(query, input.UserID, input.Content, input.ImageURL, input.IsAnonymous, status)
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	msg := "Post submitted for review!"
	if status == "approved" {
		msg = "Post published!"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: msg})
}

func handleGetPublicPosts(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	currentUserID := 0
	if cookie, err := r.Cookie("session_user_id"); err == nil {
		currentUserID, _ = strconv.Atoi(cookie.Value)
	}

	baseQuery := `
		SELECT p.id, p.user_id, p.content, COALESCE(p.image_url, ''), p.is_anonymous, p.status, p.created_at,
		       CASE WHEN p.is_anonymous THEN 'Anonymous' ELSE u.nickname END AS author_name,
		       CASE WHEN p.is_anonymous THEN 'user' ELSE u.role END AS author_role,
		       CASE WHEN p.is_anonymous THEN 'https://api.dicebear.com/7.x/bottts/svg?seed=anonymous' ELSE COALESCE(u.avatar_url, '') END AS author_pic
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.status = 'approved'
	`

	var rows *sql.Rows
	var err error

	if searchQuery != "" {
		baseQuery += ` AND (LOWER(p.content) LIKE LOWER($1) OR LOWER(u.nickname) LIKE LOWER($1) OR LOWER(u.username) LIKE LOWER($1))`
		baseQuery += ` ORDER BY p.created_at DESC`
		rows, err = db.Query(baseQuery, "%"+searchQuery+"%")
	} else {
		baseQuery += ` ORDER BY p.created_at DESC`
		rows, err = db.Query(baseQuery)
	}

	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.UserID, &p.Content, &p.ImageURL, &p.IsAnonymous, &p.Status, &p.CreatedAt, &p.AuthorName, &p.AuthorRole, &p.AuthorPic)

		p.Reactions = fetchReactions("post_id", p.ID, currentUserID)

		cRows, cErr := db.Query(`
			SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, u.nickname, u.role, COALESCE(u.avatar_url, '')
			FROM comments c
			JOIN users u ON c.user_id = u.id
			WHERE c.post_id = $1 AND c.parent_id IS NULL AND c.status = 'approved'
			ORDER BY c.created_at ASC
		`, p.ID)

		if cErr == nil {
			var comments []Comment
			for cRows.Next() {
				var c Comment
				cRows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.AuthorName, &c.AuthorRole, &c.AuthorPic)
				c.Reactions = fetchReactions("comment_id", c.ID, currentUserID)
				c.Replies = fetchCommentReplies(c.ID, currentUserID)
				comments = append(comments, c)
			}
			cRows.Close()
			p.Comments = comments
		}

		posts = append(posts, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func handleGetPendingComments(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.content, c.created_at, u.nickname, u.role, COALESCE(u.avatar_url, '')
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.status = 'pending'
		ORDER BY c.created_at ASC
	`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		rows.Scan(&c.ID, &c.PostID, &c.Content, &c.CreatedAt, &c.AuthorName, &c.AuthorRole, &c.AuthorPic)
		comments = append(comments, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func handleReviewComment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CommentID int    `json:"comment_id"`
		Status    string `json:"status"` // 'approved' or 'rejected'
	}
	json.NewDecoder(r.Body).Decode(&input)

	adminID := 0
	if cookie, err := r.Cookie("session_user_id"); err == nil {
		adminID, _ = strconv.Atoi(cookie.Value)
	}

	var commentUserID, postID int
	db.QueryRow(`SELECT user_id, post_id FROM comments WHERE id = $1`, input.CommentID).Scan(&commentUserID, &postID)

	if input.Status == "approved" {
		db.Exec(`UPDATE comments SET status = 'approved' WHERE id = $1`, input.CommentID)
		if commentUserID != adminID && adminID > 0 {
			db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'comment_approved', $3, 'Your comment was approved by an admin!')`, commentUserID, adminID, postID)
		}
	} else {
		// Send rejection notification before deleting
		if commentUserID != adminID && adminID > 0 {
			db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'comment_rejected', $3, 'Your comment was reviewed and rejected by an admin.')`, commentUserID, adminID, postID)
		}
		db.Exec(`DELETE FROM comments WHERE id = $1`, input.CommentID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Comment review updated!"})
}

func fetchCommentReplies(parentID int, currentUserID int) []Comment {
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.parent_id, c.content, c.created_at, u.nickname, u.role, COALESCE(u.avatar_url, '')
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.parent_id = $1 AND c.status = 'approved'
		ORDER BY c.created_at ASC
	`, parentID)
	if err != nil {
		return []Comment{}
	}
	defer rows.Close()

	var replies []Comment
	for rows.Next() {
		var c Comment
		rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.ParentID, &c.Content, &c.CreatedAt, &c.AuthorName, &c.AuthorRole, &c.AuthorPic)
		c.Reactions = fetchReactions("comment_id", c.ID, currentUserID)
		replies = append(replies, c)
	}
	return replies
}

func fetchReactions(targetCol string, targetID int, currentUserID int) []Reaction {
	query := fmt.Sprintf(`
		SELECT 
			emoji, 
			COUNT(*)::int AS count, 
			(COUNT(CASE WHEN user_id = $2 THEN 1 END) > 0)::boolean AS user_reacted
		FROM reactions
		WHERE %s = $1
		GROUP BY emoji
	`, targetCol)

	rows, err := db.Query(query, targetID, currentUserID)
	if err != nil {
		log.Printf("Error querying reactions for %s=%d: %v\n", targetCol, targetID, err)
		return []Reaction{}
	}
	defer rows.Close()

	var list []Reaction
	for rows.Next() {
		var r Reaction
		if err := rows.Scan(&r.Emoji, &r.Count, &r.UserReacted); err != nil {
			log.Printf("Error scanning reaction row: %v\n", err)
			continue
		}
		list = append(list, r)
	}
	return list
}

func handleEditPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID  int    `json:"user_id"`
		PostID  int    `json:"post_id"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var status string
	var authorID int
	db.QueryRow(`SELECT status, user_id FROM posts WHERE id = $1`, input.PostID).Scan(&status, &authorID)

	var requesterRole string
	db.QueryRow(`SELECT role FROM users WHERE id = $1`, input.UserID).Scan(&requesterRole)

	// Allow developer, admin, and moderator roles to bypass restrictions on approved posts
	if status == "approved" && requesterRole != "developer" && requesterRole != "admin" && requesterRole != "moderator" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Status: "error", Message: "Approved posts cannot be edited by users."})
		return
	}

	_, err := db.Exec(`UPDATE posts SET content = $1 WHERE id = $2`, input.Content, input.PostID)
	if err != nil {
		http.Error(w, "Edit failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Post updated!"})
}

func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID int `json:"user_id"`
		PostID int `json:"post_id"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var status string
	var authorID int
	db.QueryRow(`SELECT status, user_id FROM posts WHERE id = $1`, input.PostID).Scan(&status, &authorID)

	var requesterRole string
	db.QueryRow(`SELECT role FROM users WHERE id = $1`, input.UserID).Scan(&requesterRole)

	// Allow developer, admin, and moderator roles to bypass restrictions on approved posts
	if status == "approved" && requesterRole != "developer" && requesterRole != "admin" && requesterRole != "moderator" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Status: "error", Message: "Approved posts cannot be deleted by users. Please contact an admin."})
		return
	}

	_, err := db.Exec(`DELETE FROM posts WHERE id = $1`, input.PostID)
	if err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Post deleted!"})
}

func handleToggleReaction(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID    int    `json:"user_id"`
		PostID    *int   `json:"post_id,omitempty"`
		CommentID *int   `json:"comment_id,omitempty"`
		Emoji     string `json:"emoji"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var count int
	if input.PostID != nil {
		db.QueryRow(`SELECT COUNT(*) FROM reactions WHERE post_id = $1 AND user_id = $2 AND emoji = $3`, *input.PostID, input.UserID, input.Emoji).Scan(&count)
		if count > 0 {
			db.Exec(`DELETE FROM reactions WHERE post_id = $1 AND user_id = $2 AND emoji = $3`, *input.PostID, input.UserID, input.Emoji)
		} else {
			db.Exec(`INSERT INTO reactions (post_id, user_id, emoji) VALUES ($1, $2, $3)`, *input.PostID, input.UserID, input.Emoji)
			var targetUserID int
			db.QueryRow(`SELECT user_id FROM posts WHERE id = $1`, *input.PostID).Scan(&targetUserID)
			if targetUserID != input.UserID {
				db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'react_post', $3, 'reacted to your post')`, targetUserID, input.UserID, *input.PostID)
			}
		}
	} else if input.CommentID != nil {
		db.QueryRow(`SELECT COUNT(*) FROM reactions WHERE comment_id = $1 AND user_id = $2 AND emoji = $3`, *input.CommentID, input.UserID, input.Emoji).Scan(&count)
		if count > 0 {
			db.Exec(`DELETE FROM reactions WHERE comment_id = $1 AND user_id = $2 AND emoji = $3`, *input.CommentID, input.UserID, input.Emoji)
		} else {
			db.Exec(`INSERT INTO reactions (comment_id, user_id, emoji) VALUES ($1, $2, $3)`, *input.CommentID, input.UserID, input.Emoji)
			var targetUserID int
			var postID int
			db.QueryRow(`SELECT user_id, post_id FROM comments WHERE id = $1`, *input.CommentID).Scan(&targetUserID, &postID)
			if targetUserID != input.UserID {
				db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'react_comment', $3, 'reacted to your comment')`, targetUserID, input.UserID, postID)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success"})
}

func handleGetReactionUsers(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("post_id")
	commentID := r.URL.Query().Get("comment_id")

	var rows *sql.Rows
	var err error

	if postID != "" {
		rows, err = db.Query(`
			SELECT u.username, u.nickname, COALESCE(u.avatar_url, ''), r.emoji
			FROM reactions r
			JOIN users u ON r.user_id = u.id
			WHERE r.post_id = $1
		`, postID)
	} else if commentID != "" {
		rows, err = db.Query(`
			SELECT u.username, u.nickname, COALESCE(u.avatar_url, ''), r.emoji
			FROM reactions r
			JOIN users u ON r.user_id = u.id
			WHERE r.comment_id = $1
		`, commentID)
	}

	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []ReactionUser
	for rows.Next() {
		var ru ReactionUser
		rows.Scan(&ru.Username, &ru.Nickname, &ru.AvatarURL, &ru.Emoji)
		list = append(list, ru)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PostID   int    `json:"post_id"`
		UserID   int    `json:"user_id"`
		ParentID *int   `json:"parent_id,omitempty"`
		Content  string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	var role string
	db.QueryRow(`SELECT role FROM users WHERE id = $1`, input.UserID).Scan(&role)

	status := "pending"
	// Developer, Admin, and Moderator comments bypass moderation and are auto-approved
	if role == "developer" || role == "admin" || role == "moderator" {
		status = "approved"
	}

	query := `INSERT INTO comments (post_id, user_id, parent_id, content, status) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(query, input.PostID, input.UserID, input.ParentID, input.Content, status)
	if err != nil {
		http.Error(w, "Failed to comment", http.StatusInternalServerError)
		return
	}

	msg := "Comment submitted for review!"
	if status == "approved" {
		msg = "Comment added!"
		if input.ParentID != nil {
			var parentUserID int
			db.QueryRow(`SELECT user_id FROM comments WHERE id = $1`, *input.ParentID).Scan(&parentUserID)
			if parentUserID != input.UserID {
				db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'reply_comment', $3, 'replied to your comment')`, parentUserID, input.UserID, input.PostID)
			}
		} else {
			var postUserID int
			db.QueryRow(`SELECT user_id FROM posts WHERE id = $1`, input.PostID).Scan(&postUserID)
			if postUserID != input.UserID {
				db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'comment_post', $3, 'commented on your post')`, postUserID, input.UserID, input.PostID)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: msg})
}

func handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CommentID int `json:"comment_id"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	_, err := db.Exec(`DELETE FROM comments WHERE id = $1`, input.CommentID)
	if err != nil {
		http.Error(w, "Delete comment failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Comment deleted!"})
}

func handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	userID := 0
	if cookie, err := r.Cookie("session_user_id"); err == nil && cookie.Value != "" {
		userID, _ = strconv.Atoi(cookie.Value)
	}

	if userID == 0 {
		paramID := r.URL.Query().Get("user_id")
		userID, _ = strconv.Atoi(paramID)
	}

	rows, err := db.Query(`
		SELECT n.id, u.nickname AS actor_name, COALESCE(u.avatar_url, '') AS actor_pic, n.type, n.post_id, n.message, n.is_read, n.created_at
		FROM notifications n
		JOIN users u ON n.actor_id = u.id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC
		LIMIT 20
	`, userID)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []Notification
	for rows.Next() {
		var n Notification
		rows.Scan(&n.ID, &n.ActorName, &n.ActorPic, &n.Type, &n.PostID, &n.Message, &n.IsRead, &n.CreatedAt)
		list = append(list, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func handleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_user_id")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(cookie.Value)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Update all unread notifications for this user to true
	_, err = db.Exec(`UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`, userID)
	if err != nil {
		log.Printf("Error marking notifications read for user %d: %v\n", userID, err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Notifications marked as read"})
}

func handleGetUsersList(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, username, nickname, email, role, is_banned, COALESCE(avatar_url, '') FROM users ORDER BY id ASC`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Username, &u.Nickname, &u.Email, &u.Role, &u.IsBanned, &u.AvatarURL)
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID int    `json:"user_id"`
		Role   string `json:"role"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	db.Exec(`UPDATE users SET role = $1 WHERE id = $2`, input.Role, input.UserID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "User role updated!"})
}

func handleBanUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID   int  `json:"user_id"`
		IsBanned bool `json:"is_banned"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	db.Exec(`UPDATE users SET is_banned = $1 WHERE id = $2`, input.IsBanned, input.UserID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "User ban status updated!"})
}

func handleGetPendingPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT p.id, p.content, COALESCE(p.image_url, ''), p.is_anonymous, p.created_at, 
		       CASE 
		           WHEN p.is_anonymous THEN u.nickname || ' (Anonymous Post)' 
		           ELSE u.nickname 
		       END AS author_name, 
		       u.role AS author_role, COALESCE(u.avatar_url, '') AS author_pic
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.status = 'pending'
		ORDER BY p.created_at ASC
	`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.Content, &p.ImageURL, &p.IsAnonymous, &p.CreatedAt, &p.AuthorName, &p.AuthorRole, &p.AuthorPic)
		posts = append(posts, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func handleReviewPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PostID int    `json:"post_id"`
		Status string `json:"status"` // 'approved' or 'rejected'
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	adminID := 0
	if cookie, err := r.Cookie("session_user_id"); err == nil {
		adminID, _ = strconv.Atoi(cookie.Value)
	}

	var postUserID int
	err := db.QueryRow(`SELECT user_id FROM posts WHERE id = $1`, input.PostID).Scan(&postUserID)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	if input.Status == "approved" {
		db.Exec(`UPDATE posts SET status = 'approved' WHERE id = $1`, input.PostID)
		if postUserID != adminID && adminID > 0 {
			db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'post_approved', $3, 'Your post was approved by an admin!')`, postUserID, adminID, input.PostID)
		}
	} else {
		// 1. Send rejection notification without tying post_id (since post will be deleted)
		if postUserID != adminID && adminID > 0 {
			db.Exec(`INSERT INTO notifications (user_id, actor_id, type, post_id, message) VALUES ($1, $2, 'post_rejected', NULL, 'Your post was reviewed and rejected by an admin.')`, postUserID, adminID)
		}

		// 2. Clear related child records so foreign keys don't block post deletion
		db.Exec(`DELETE FROM reactions WHERE post_id = $1`, input.PostID)
		db.Exec(`DELETE FROM comments WHERE post_id = $1`, input.PostID)

		// 3. Delete the post cleanly
		_, delErr := db.Exec(`DELETE FROM posts WHERE id = $1`, input.PostID)
		if delErr != nil {
			log.Printf("Error deleting rejected post #%d: %v\n", input.PostID, delErr)
			http.Error(w, "Failed to reject post", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Post review updated!"})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if cld == nil {
		http.Error(w, "Cloudinary unconfigured", http.StatusInternalServerError)
		return
	}

	r.ParseMultipartForm(10 << 20)
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed file read", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ctx := context.Background()
	uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{Folder: "corstellae_uploads"})
	if err != nil {
		http.Error(w, "Cloudinary upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", URL: uploadResult.SecureURL})
}
