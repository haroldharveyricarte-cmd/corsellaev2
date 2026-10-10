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

type Comment struct {
	ID         int    `json:"id"`
	PostID     int    `json:"post_id"`
	UserID     int    `json:"user_id"`
	AuthorName string `json:"author_name"`
	AuthorRole string `json:"author_role"`
	AuthorPic  string `json:"author_pic"`
	Content    string `json:"content"`
	CreatedAt  string `json:"created_at"`
}

type Post struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	AuthorName  string    `json:"author_name"`
	AuthorRole  string    `json:"author_role"`
	AuthorPic   string    `json:"author_pic"`
	Content     string    `json:"content"`
	ImageURL    string    `json:"image_url"`
	IsAnonymous bool      `json:"is_anonymous"`
	Status      string    `json:"status"`
	CreatedAt   string    `json:"created_at"`
	Comments    []Comment `json:"comments"`
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
	var adminID int
	err := db.QueryRow(`SELECT id FROM users WHERE username = 'gabby'`).Scan(&adminID)
	if err == sql.ErrNoRows {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("gabbytheowner"), bcrypt.DefaultCost)
		_, err := db.Exec(`INSERT INTO users (username, nickname, email, password, role) VALUES ('gabby', 'Gabby (Owner)', 'gabby@corstellae.com', $1, 'admin')`, string(hashed))
		if err == nil {
			fmt.Println("👑 Hardcoded Admin Account Initialized: gabby / gabbytheowner")
		}
	} else {
		db.Exec(`UPDATE users SET role = 'admin' WHERE username = 'gabby'`)
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

	// Page Routes
	http.HandleFunc("/", serveLogin)
	http.HandleFunc("/dashboard", serveDashboard)

	// Auth & User Profile APIs
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

	// Comment APIs
	http.HandleFunc("/api/comments/create", handleCreateComment)
	http.HandleFunc("/api/comments/delete", handleDeleteComment)

	// Admin / Moderator APIs
	http.HandleFunc("/api/admin/users", handleGetUsersList)
	http.HandleFunc("/api/admin/user/role", handleUpdateUserRole)
	http.HandleFunc("/api/admin/user/ban", handleBanUser)
	http.HandleFunc("/api/admin/pending", handleGetPendingPosts)
	http.HandleFunc("/api/admin/review", handleReviewPost)

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

	// Auto-redirect to dashboard if session cookie is valid
	if cookie, err := r.Cookie("session_user_id"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	http.ServeFile(w, r, "./templates/login.html")
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	// Authenticate 7-day persistent cookie
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

	// 7-Day Persistent Session Cookie
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
	if role == "admin" || role == "moderator" {
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
	// Updated Query: When is_anonymous is TRUE, hide user nickname, avatar, and staff role
	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.content, COALESCE(p.image_url, ''), p.is_anonymous, p.created_at,
		       CASE WHEN p.is_anonymous THEN 'Anonymous' ELSE u.nickname END AS author_name,
		       CASE WHEN p.is_anonymous THEN 'user' ELSE u.role END AS author_role,
		       CASE WHEN p.is_anonymous THEN 'https://api.dicebear.com/7.x/bottts/svg?seed=anonymous' ELSE COALESCE(u.avatar_url, '') END AS author_pic
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.status = 'approved'
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		rows.Scan(&p.ID, &p.UserID, &p.Content, &p.ImageURL, &p.IsAnonymous, &p.CreatedAt, &p.AuthorName, &p.AuthorRole, &p.AuthorPic)

		cRows, cErr := db.Query(`
			SELECT c.id, c.post_id, c.user_id, c.content, c.created_at, u.nickname, u.role, COALESCE(u.avatar_url, '')
			FROM comments c
			JOIN users u ON c.user_id = u.id
			WHERE c.post_id = $1
			ORDER BY c.created_at ASC
		`, p.ID)

		if cErr == nil {
			var comments []Comment
			for cRows.Next() {
				var c Comment
				cRows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.AuthorName, &c.AuthorRole, &c.AuthorPic)
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

func handleEditPost(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PostID  int    `json:"post_id"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&input)

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
		PostID int `json:"post_id"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	_, err := db.Exec(`DELETE FROM posts WHERE id = $1`, input.PostID)
	if err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Post deleted!"})
}

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PostID  int    `json:"post_id"`
		UserID  int    `json:"user_id"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	_, err := db.Exec(`INSERT INTO comments (post_id, user_id, content) VALUES ($1, $2, $3)`, input.PostID, input.UserID, input.Content)
	if err != nil {
		http.Error(w, "Failed to comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Status: "success", Message: "Comment added!"})
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
		SELECT p.id, p.content, COALESCE(p.image_url, ''), p.is_anonymous, p.created_at, u.nickname AS author_name, u.role AS author_role, COALESCE(u.avatar_url, '') AS author_pic
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
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&input)

	db.Exec(`UPDATE posts SET status = $1 WHERE id = $2`, input.Status, input.PostID)
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
