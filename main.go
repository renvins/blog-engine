package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
)

//go:embed content/*.md
var contentFS embed.FS

//go:embed templates/*.html
var templatesFS embed.FS

func main() {
	// Load posts into RAM
	loadPosts()

	// Route: homepage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		tmpl, err := template.ParseFS(templatesFS, "templates/layout.html", "templates/index.html")
		if err != nil {
			http.Error(w, "Template Error: "+err.Error(), 500)
			return
		}

		data := IndexData{
			Title: "renvins' thoughts blog",
			Posts: postCache,
		}

		tmpl.Execute(w, data)
	})

	// Route: single post
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		// Extract slug
		slug := strings.TrimPrefix(r.URL.Path, "/post/")

		var foundPost Post
		found := false
		for _, p := range postCache {
			if p.Slug == slug {
				foundPost = p
				found = true
				break
			}
		}

		if !found {
			http.NotFound(w, r)
			return
		}

		// Parse layout + post
		tmpl := template.Must(template.ParseFiles("templates/layout.html", "templates/post.html"))

		data := PostData{
			Title: foundPost.Title,
			Post:  foundPost,
		}

		tmpl.Execute(w, data)
	})

	fmt.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Post represents a single blog entry
type Post struct {
	Title   string
	Date    time.Time
	Slug    string        // The URL path
	Content template.HTML // Pre-rendered HTML content
	Summary string        // For the homepage preview
}

// Global cache to store posts in RAM
var postCache []Post

func loadPosts() {
	files, err := contentFS.ReadDir("content")
	if err != nil {
		log.Fatal(err)
	}

	var posts []Post

	for _, fileEntry := range files {
		// Read the raw file
		content, err := contentFS.ReadFile("content/" + fileEntry.Name())
		if err != nil {
			log.Println("Error reading file:", err)
			continue
		}

		// Parse the frontmatter (metadata)
		// We assume the file starts with "---", metadata, "---"
		parts := strings.SplitN(string(content), "---", 3)
		if len(parts) < 3 {
			log.Println("Error parsing file:", fileEntry.Name())
			continue
		}

		metaRaw := parts[1]
		bodyRaw := parts[2]

		// Extract metadata manually
		post := Post{}
		lines := strings.Split(metaRaw, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Title: ") {
				post.Title = strings.TrimPrefix(line, "Title: ")
			}
			if strings.HasPrefix(line, "Date: ") {
				dateStr := strings.TrimPrefix(line, "Date: ")
				post.Date, _ = time.Parse("2006-01-02", strings.TrimSpace(dateStr))
			}
		}

		// Generate the slug from the filename
		filename := fileEntry.Name()
		post.Slug = strings.TrimSuffix(filename, filepath.Ext(filename))

		// Convert markdown body to HTML using Goldmark
		var buf bytes.Buffer
		if err := goldmark.Convert([]byte(bodyRaw), &buf); err != nil {
			log.Println("Error converting goldmark:", err)
			continue
		}
		post.Content = template.HTML(buf.String())

		if len(bodyRaw) > 150 {
			post.Summary = bodyRaw[:150] + "..."
		} else {
			post.Summary = bodyRaw
		}

		posts = append(posts, post)
	}

	// Sort by date
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	postCache = posts
	fmt.Println("Loaded posts:", len(postCache))
}

// Data passed to the index template
type IndexData struct {
	Title string
	Posts []Post
}

// Data passed to the post template
type PostData struct {
	Title string
	Post  Post
}
