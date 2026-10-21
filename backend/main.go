package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed frontend/dist data/questions-2025.json
var assets embed.FS

type server struct {
	bank     QuestionBank
	sessions map[string]*Session
	mu       sync.Mutex
	store    *dataStore
	storeErr error
}

func loadBank() (QuestionBank, error) {
	var b QuestionBank
	raw, err := assets.ReadFile("data/questions-2025.json")
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(raw, &b)
	if err != nil {
		return b, err
	}
	return b, validateBank(b)
}
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *server) persist() error {
	if s.storeErr != nil {
		return s.storeErr
	}
	return s.store.save()
}
func (s *server) api(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.storeErr != nil {
		writeJSON(w, 503, map[string]string{"error": "Your saved history could not be opened. It has not been changed: " + s.storeErr.Error()})
		return
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/api/metadata":
		writeJSON(w, 200, map[string]any{"revision": s.bank.Revision, "total": len(s.bank.Questions), "officialSource": s.bank.OfficialSource, "updatesSource": s.bank.UpdatesSource})
		return
	case r.Method == "GET" && r.URL.Path == "/api/current":
		writeJSON(w, 200, s.store.data.Current)
		return
	case r.Method == "GET" && r.URL.Path == "/api/history":
		writeJSON(w, 200, s.store.data.History)
		return
	case r.Method == "GET" && r.URL.Path == "/api/preferences":
		writeJSON(w, 200, s.store.data.Preferences)
		return
	case r.Method == "PUT" && r.URL.Path == "/api/preferences":
		var p Preferences
		if json.NewDecoder(r.Body).Decode(&p) != nil || (p.Theme != "light" && p.Theme != "dark" && p.Theme != "system") || (p.PracticeMode != "multiple-choice" && p.PracticeMode != "oral") || p.DefaultQuizSize < 1 || p.DefaultQuizSize > 128 {
			writeJSON(w, 400, map[string]string{"error": "invalid preferences"})
			return
		}
		s.store.data.Preferences = p
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, p)
		return
	case r.Method == "POST" && r.URL.Path == "/api/quizzes":
		var in struct {
			Order string `json:"order"`
			Mode  string `json:"mode"`
			Count int    `json:"count"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		q, err := makeSession(s.bank, in.Order, in.Mode, in.Count)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		s.mu.Lock()
		s.sessions[q.ID] = q
		s.store.data.Current = q
		s.mu.Unlock()
		if err = s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, q)
		return
	}
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 || parts[1] != "quizzes" {
		writeJSON(w, 404, map[string]string{"error": "not found"})
		return
	}
	s.mu.Lock()
	session := s.sessions[parts[2]]
	s.mu.Unlock()
	if session == nil {
		writeJSON(w, 404, map[string]string{"error": "quiz not found"})
		return
	}
	if r.Method == "GET" && len(parts) == 3 {
		writeJSON(w, 200, session)
		return
	}
	if r.Method == "PUT" && len(parts) == 5 && parts[3] == "responses" {
		var in struct {
			OptionIDs []string `json:"optionIds"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		var id int
		_, _ = fmt.Sscanf(parts[4], "%d", &id)
		if err := updateResponse(session, id, in.OptionIDs); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"responses": session.Responses})
		return
	}
	if r.Method == "POST" && len(parts) == 5 && parts[3] == "reveal" {
		var id int
		_, _ = fmt.Sscanf(parts[4], "%d", &id)
		if err := reveal(session, id); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"revealed": session.Revealed})
		return
	}
	if r.Method == "PUT" && len(parts) == 5 && parts[3] == "self-grade" {
		var in struct {
			Correct bool `json:"correct"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid request"})
			return
		}
		var id int
		_, _ = fmt.Sscanf(parts[4], "%d", &id)
		if err := selfGrade(session, id, in.Correct); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"selfGrades": session.SelfGrades})
		return
	}
	if r.Method == "POST" && len(parts) == 4 && parts[3] == "submit" {
		result, err := grade(session)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		order := make([]int, len(session.Questions))
		for i, q := range session.Questions {
			order[i] = q.ID
		}
		s.store.data.History = append([]Attempt{{ID: session.ID, CompletedAt: session.Created, Mode: session.Mode, QuestionCount: len(order), QuestionOrder: order, BankRevision: s.bank.Revision, Result: result}}, s.store.data.History...)
		s.store.data.Current = nil
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, result)
		return
	}
	writeJSON(w, 404, map[string]string{"error": "not found"})
}
func splitPath(p string) []string {
	out := []string{}
	for _, x := range strings.Split(p, "/") {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}
func serverLockPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "USCivicsPractice", "server.lock"), nil
}

// acquireServerLock prevents a second copy from opening another browser tab.
// A stale lock is removed only after its recorded local address cannot be reached.
func acquireServerLock() (func(), error) {
	path, err := serverLockPath()
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		f, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if createErr == nil {
			_, _ = f.WriteString("starting")
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(createErr, os.ErrExist) {
			return nil, createErr
		}
		raw, readErr := os.ReadFile(path)
		address := strings.TrimSpace(string(raw))
		if address == "starting" {
			info, statErr := os.Stat(path)
			if statErr == nil && time.Since(info.ModTime()) < 15*time.Second {
				return nil, errors.New("US Civics Practice is already starting; please wait a moment")
			}
		}
		if readErr == nil && address != "" {
			conn, dialErr := net.DialTimeout("tcp", address, 500*time.Millisecond)
			if dialErr == nil {
				_ = conn.Close()
				return nil, fmt.Errorf("US Civics Practice is already running at http://%s", address)
			}
		}
		if removeErr := os.Remove(path); removeErr != nil {
			return nil, fmt.Errorf("could not clear stale server lock: %w", removeErr)
		}
	}
	return nil, errors.New("could not acquire application startup lock")
}

func openBrowser(url string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}

func main() {
	b, err := loadBank()
	if err != nil {
		log.Fatal("Could not load the built-in question bank: ", err)
	}
	unlock, err := acquireServerLock()
	if err != nil {
		log.Fatal("Could not start US Civics Practice: ", err)
	}
	defer unlock()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Could not open a local-only server: ", err)
	}
	defer listener.Close()
	address := listener.Addr().String()
	lockPath, _ := serverLockPath()
	if err := os.WriteFile(lockPath, []byte(address), 0600); err != nil {
		log.Fatal("Could not record application startup: ", err)
	}

	st, storeErr := loadStore()
	if storeErr != nil {
		log.Printf("History unavailable: %v", storeErr)
	}
	if st == nil {
		st = &dataStore{data: AppData{SchemaVersion: dataSchemaVersion, Preferences: defaultPreferences()}}
	}
	s := &server{bank: b, sessions: map[string]*Session{}, store: st, storeErr: storeErr}
	if st.data.Current != nil && !st.data.Current.Submitted {
		s.sessions[st.data.Current.ID] = st.data.Current
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.api)
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal("Could not load built-in web assets: ", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	url := "http://" + address
	log.Printf("US Civics Practice is running at %s", url)
	log.Println("Close this window or press Ctrl+C when you are finished studying.")
	if err := openBrowser(url); err != nil {
		log.Printf("Could not open your browser automatically. Open %s instead: %v", url, err)
	}
	if err := http.Serve(listener, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Local server stopped unexpectedly: ", err)
	}
}
