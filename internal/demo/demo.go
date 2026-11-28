// Package demo builds a small repository whose history hides a handful of
// security fixes among ordinary work — some announced, most not — so Culprit
// has something real to find in its tests and its demo stack.
package demo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type step struct {
	day     int // days after the first commit
	author  string
	message string
	files   map[string]string
}

const frameV1 = `package decode

import "encoding/binary"

// Frame reads one length-prefixed record.
func Frame(buf []byte) []byte {
	n := int(binary.BigEndian.Uint16(buf[0:2]))
	return buf[2 : 2+n]
}
`

const frameV2 = `package decode

import "encoding/binary"

// Version is the wire format revision.
const Version = 2

// Frame reads one length-prefixed record.
func Frame(buf []byte) []byte {
	n := int(binary.BigEndian.Uint16(buf[0:2]))
	return buf[2 : 2+n]
}
`

const frameV3 = `package decode

import (
	"encoding/binary"
	"errors"
)

// Version is the wire format revision.
const Version = 2

var errShort = errors.New("short frame")

// Frame reads one length-prefixed record.
func Frame(buf []byte) ([]byte, error) {
	if len(buf) < 2 {
		return nil, errShort
	}
	n := int(binary.BigEndian.Uint16(buf[0:2]))
	if 2+n > len(buf) {
		return nil, errShort
	}
	return buf[2 : 2+n], nil
}
`

const sessionV1 = `package auth

// Valid reports whether the presented token matches the stored one.
func Valid(presented, stored string) bool {
	return presented == stored
}
`

const sessionV2 = `package auth

import "time"

// Session is a logged-in user.
type Session struct {
	Token   string
	Expires time.Time
}

// Valid reports whether the presented token matches the stored one.
func Valid(presented string, s Session) bool {
	if time.Now().After(s.Expires) {
		return false
	}
	return presented == s.Token
}
`

const sessionV3 = `package auth

import (
	"crypto/subtle"
	"time"
)

// Session is a logged-in user.
type Session struct {
	Token   string
	Expires time.Time
}

// Valid reports whether the presented token matches the stored one.
func Valid(presented string, s Session) bool {
	if time.Now().After(s.Expires) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(s.Token)) == 1
}
`

const serveV1 = `package files

import (
	"net/http"
	"path"
)

// Handler serves files from root.
func Handler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(root, r.URL.Query().Get("name")))
	}
}
`

const serveV2 = `package files

import (
	"net/http"
	"path/filepath"
	"strings"
)

// Handler serves files from root.
func Handler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := filepath.Clean(r.URL.Query().Get("name"))
		if strings.Contains(name, "..") {
			http.Error(w, "bad name", http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, name))
	}
}
`

const usersV1 = `package store

import (
	"database/sql"
	"fmt"
)

// Find looks a user up by name.
func Find(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query(fmt.Sprintf("SELECT id, name FROM users WHERE name = '%s'", name))
}
`

const usersV2 = `package store

import (
	"database/sql"
	"fmt"
)

// Find looks a user up by name.
func Find(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query(fmt.Sprintf("SELECT id, name FROM users WHERE name = '%s'", name))
}

// List returns one page of users.
func List(db *sql.DB, limit, offset int) (*sql.Rows, error) {
	return db.Query("SELECT id, name FROM users ORDER BY id LIMIT $1 OFFSET $2", limit, offset)
}
`

const usersV3 = `package store

import "database/sql"

// Find looks a user up by name.
func Find(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query("SELECT id, name FROM users WHERE name = $1", name)
}

// List returns one page of users.
func List(db *sql.DB, limit, offset int) (*sql.Rows, error) {
	return db.Query("SELECT id, name FROM users ORDER BY id LIMIT $1 OFFSET $2", limit, offset)
}
`

const uploadV1 = `package api

import (
	"io"
	"net/http"
	"os"
)

// Upload stores the request body.
func Upload(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = os.WriteFile("upload.bin", data, 0o600)
}
`

const uploadV2 = `package api

import (
	"io"
	"net/http"
	"os"
)

const maxUpload = 8 << 20

// Upload stores the request body.
func Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}
	_ = os.WriteFile("upload.bin", data, 0o600)
}
`

const configV1 = `package config

// Config is the service configuration.
type Config struct {
	Addr string
	DB   string
}

// Load returns the defaults.
func Load() *Config { return &Config{Addr: ":8080"} }

// Addr returns the listen address.
func Addr(c *Config) string { return c.Addr }
`

const configV2 = `package config

// Config is the service configuration.
type Config struct {
	Addr string
	DB   string
}

// Load returns the defaults.
func Load() *Config { return &Config{Addr: ":8080"} }

// Addr returns the listen address.
func Addr(c *Config) string {
	if c == nil {
		return ":8080"
	}
	return c.Addr
}
`

var history = []step{
	{0, "Ada Park", "initial commit", map[string]string{"README.md": "# relay\n\nA small file and message relay.\n"}},
	{3, "Ada Park", "add the frame decoder", map[string]string{"decode/frame.go": frameV1}},
	{6, "Ben Ito", "add session authentication", map[string]string{"auth/session.go": sessionV1}},
	{10, "Ben Ito", "serve stored files", map[string]string{"files/serve.go": serveV1}},
	{14, "Cleo Ruiz", "user store", map[string]string{"store/users.go": usersV1}},
	{17, "Cleo Ruiz", "accept uploads", map[string]string{"api/upload.go": uploadV1}},
	{21, "Ada Park", "configuration loading", map[string]string{"config/config.go": configV1}},
	{25, "Ada Park", "docs: usage section", map[string]string{"README.md": "# relay\n\nA small file and message relay.\n\n## Usage\n\n    relay -addr :8080\n"}},
	{31, "Ben Ito", "add a wire format version", map[string]string{"decode/frame.go": frameV2}},
	{38, "Cleo Ruiz", "paginate the user list", map[string]string{"store/users.go": usersV2}},
	{44, "Ada Park", "tidy frame reader", map[string]string{"decode/frame.go": frameV3}},
	{50, "Ben Ito", "sessions expire", map[string]string{"auth/session.go": sessionV2}},
	{56, "Cleo Ruiz", "docs: changelog", map[string]string{"CHANGELOG.md": "# Changelog\n\n## 0.2.0\n- pagination\n- session expiry\n"}},
	{63, "Ben Ito", "refactor session check", map[string]string{"auth/session.go": sessionV3}},
	{70, "Ada Park", "small fix", map[string]string{"files/serve.go": serveV2}},
	{77, "Cleo Ruiz", "bump version to 0.3.0", map[string]string{"VERSION": "0.3.0\n"}},
	{85, "Cleo Ruiz", "fix SQL injection in user lookup (CVE-2025-31337)", map[string]string{"store/users.go": usersV3}},
	{92, "Ben Ito", "limit upload size", map[string]string{"api/upload.go": uploadV2}},
	{99, "Ada Park", "docs: contributing guide", map[string]string{"CONTRIBUTING.md": "# Contributing\n\nOpen a pull request.\n"}},
	{106, "Ada Park", "fix", map[string]string{"config/config.go": configV2}},
	{112, "Cleo Ruiz", "release 0.4.0", map[string]string{"VERSION": "0.4.0\n", "CHANGELOG.md": "# Changelog\n\n## 0.4.0\n- upload limits\n\n## 0.2.0\n- pagination\n- session expiry\n"}},
}

// Epoch is the date of the first commit in the generated history.
var Epoch = time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)

// Build writes the demo repository into dir, which must not already hold one.
func Build(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	if err := git(ctx, dir, nil, "init", "-q", "-b", "main"); err != nil {
		return err
	}
	for _, s := range history {
		for name, content := range s.files {
			full := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
				return err
			}
			if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
				return err
			}
		}
		when := Epoch.AddDate(0, 0, s.day).Format(time.RFC3339)
		env := []string{"GIT_AUTHOR_DATE=" + when, "GIT_COMMITTER_DATE=" + when}
		if err := git(ctx, dir, nil, "add", "-A"); err != nil {
			return err
		}
		email := fmt.Sprintf("%s@relay.example", firstName(s.author))
		if err := git(ctx, dir, env, "-c", "user.name="+s.author, "-c", "user.email="+email,
			"-c", "commit.gpgsign=false", "commit", "-q", "-m", s.message); err != nil {
			return err
		}
	}
	return nil
}

// Commits is the number of commits Build writes.
func Commits() int { return len(history) }

func git(ctx context.Context, dir string, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %v: %w: %s", args, err, out)
	}
	return nil
}

func firstName(full string) string {
	for i, r := range full {
		if r == ' ' {
			return lower(full[:i])
		}
	}
	return lower(full)
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
