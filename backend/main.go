package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
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

func (s *server) question(id int) *Question {
	for i := range s.bank.Questions {
		if s.bank.Questions[i].ID == id {
			return &s.bank.Questions[i]
		}
	}
	return nil
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
		writeJSON(w, 200, s.store.active().Current)
		return
	case r.Method == "GET" && r.URL.Path == "/api/history":
		writeJSON(w, 200, s.store.active().History)
		return
	case r.Method == "GET" && r.URL.Path == "/api/preferences":
		writeJSON(w, 200, s.store.active().Preferences)
		return
	case r.Method == "GET" && r.URL.Path == "/api/profiles":
		writeJSON(w, 200, map[string]any{"activeProfileId": s.store.data.ActiveProfileID, "profiles": s.store.data.Profiles})
		return
	case r.Method == "GET" && r.URL.Path == "/api/current-answers":
		p := s.store.active()
		out := []map[string]any{}
		for id, info := range currentAnswerInfo {
			rec, ok := p.CurrentAnswers[id]
			if !ok {
				rec = CurrentAnswerRecord{QuestionID: id, Status: "Unresolved", SourceURL: info.VerificationURL}
				if a := suggestedCurrentAnswer(id, p.Preferences); len(a) > 0 {
					now := time.Now()
					rec.Suggested = a
					rec.SuggestionAt = &now
					rec.Source = "Bundled state-capital reference"
					rec.Status = "Reverification recommended"
				}
			}
			out = append(out, map[string]any{"info": info, "record": rec, "question": s.question(id)})
		}
		writeJSON(w, 200, out)
		return
	case r.Method == "POST" && r.URL.Path == "/api/current-answers/check":
		p := s.store.active()
		checked := 0
		for id := range currentAnswerInfo {
			if a := suggestedCurrentAnswer(id, p.Preferences); len(a) > 0 {
				now := time.Now()
				r := p.CurrentAnswers[id]
				r.QuestionID = id
				r.Suggested = a
				r.SuggestionAt = &now
				r.Source = "Bundled state-capital reference"
				r.SourceURL = currentAnswerInfo[id].VerificationURL
				if r.Status == "" {
					r.Status = "Reverification recommended"
				}
				p.CurrentAnswers[id] = r
				checked++
			}
		}
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"checked": checked, "message": "State-capital reference refreshed. Other answers need your review on the linked official pages."})
		return
	case r.Method == "POST" && r.URL.Path == "/api/profiles":
		var in struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
			writeJSON(w, 400, map[string]string{"error": "a profile name is required"})
			return
		}
		p := newProfile(strings.TrimSpace(in.Name))
		s.store.data.Profiles = append(s.store.data.Profiles, p)
		s.store.data.ActiveProfileID = p.ID
		_ = s.persist()
		writeJSON(w, 201, p)
		return
	case r.Method == "PUT" && r.URL.Path == "/api/profiles/active":
		var in struct {
			ID string `json:"id"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid profile"})
			return
		}
		found := false
		for i := range s.store.data.Profiles {
			if s.store.data.Profiles[i].ID == in.ID {
				s.store.data.Profiles[i].LastUsedAt = time.Now()
				found = true
			}
		}
		if !found {
			writeJSON(w, 404, map[string]string{"error": "profile not found"})
			return
		}
		s.store.data.ActiveProfileID = in.ID
		_ = s.persist()
		writeJSON(w, 200, s.store.active())
		return
	case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/api/profiles/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
		var in struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || strings.TrimSpace(in.Name) == "" {
			writeJSON(w, 400, map[string]string{"error": "a profile name is required"})
			return
		}
		for i := range s.store.data.Profiles {
			if s.store.data.Profiles[i].ID == id {
				s.store.data.Profiles[i].Name = strings.TrimSpace(in.Name)
				if err := s.persist(); err != nil {
					writeJSON(w, 500, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, 200, s.store.data.Profiles[i])
				return
			}
		}
		writeJSON(w, 404, map[string]string{"error": "profile not found"})
		return
	case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/api/profiles/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/profiles/")
		if len(s.store.data.Profiles) == 1 {
			writeJSON(w, 400, map[string]string{"error": "keep at least one profile"})
			return
		}
		next := []Profile{}
		found := false
		for _, p := range s.store.data.Profiles {
			if p.ID == id {
				found = true
			} else {
				next = append(next, p)
			}
		}
		if !found {
			writeJSON(w, 404, map[string]string{"error": "profile not found"})
			return
		}
		s.store.data.Profiles = next
		if s.store.data.ActiveProfileID == id {
			s.store.data.ActiveProfileID = next[0].ID
		}
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "deleted"})
		return
	case r.Method == "GET" && r.URL.Path == "/api/profile-export":
		writeJSON(w, 200, map[string]any{"schemaVersion": 1, "exportedAt": time.Now(), "profile": s.store.active()})
		return
	case r.Method == "POST" && r.URL.Path == "/api/profile-import":
		r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
		var in struct {
			SchemaVersion int     `json:"schemaVersion"`
			Profile       Profile `json:"profile"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || in.SchemaVersion != 1 || strings.TrimSpace(in.Profile.Name) == "" {
			writeJSON(w, 400, map[string]string{"error": "This backup is not recognized or is too large."})
			return
		}
		in.Profile.ID = newID()
		in.Profile.CreatedAt = time.Now()
		in.Profile.LastUsedAt = time.Now()
		if in.Profile.History == nil {
			in.Profile.History = []Attempt{}
		}
		if in.Profile.CurrentAnswers == nil {
			in.Profile.CurrentAnswers = map[int]CurrentAnswerRecord{}
		}
		s.store.data.Profiles = append(s.store.data.Profiles, in.Profile)
		s.store.data.ActiveProfileID = in.Profile.ID
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, in.Profile)
		return
	case r.Method == "PUT" && r.URL.Path == "/api/current-answers":
		var rec CurrentAnswerRecord
		if json.NewDecoder(r.Body).Decode(&rec) != nil || currentAnswerInfo[rec.QuestionID].QuestionID == 0 || len(rec.Answers) == 0 {
			writeJSON(w, 400, map[string]string{"error": "valid question and answer required"})
			return
		}
		info := currentAnswerInfo[rec.QuestionID]
		rec.Source = "Manually confirmed"
		rec.Status = "Manually confirmed"
		rec.SourceURL = info.VerificationURL
		now := time.Now()
		rec.VerifiedAt = &now
		s.store.active().CurrentAnswers[rec.QuestionID] = rec
		if err := s.persist(); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, rec)
		return
	case r.Method == "PUT" && r.URL.Path == "/api/preferences":
		var p Preferences
		if json.NewDecoder(r.Body).Decode(&p) != nil || (p.Theme != "light" && p.Theme != "dark" && p.Theme != "system") || (p.PracticeMode != "multiple-choice" && p.PracticeMode != "oral") || p.DefaultQuizSize < 1 || p.DefaultQuizSize > 128 {
			writeJSON(w, 400, map[string]string{"error": "invalid preferences"})
			return
		}
		s.store.active().Preferences = p
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
		q, err := makeProfileSession(s.bank, in.Order, in.Mode, in.Count, s.store.active())
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		q.ProfileID = s.store.data.ActiveProfileID
		s.mu.Lock()
		s.sessions[q.ID] = q
		s.store.active().Current = q
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
	if session.ProfileID != s.store.data.ActiveProfileID {
		writeJSON(w, 404, map[string]string{"error": "quiz belongs to another profile"})
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
		p := s.store.active()
		p.History = append([]Attempt{{ID: session.ID, CompletedAt: session.Created, Mode: session.Mode, QuestionCount: len(order), QuestionOrder: order, BankRevision: s.bank.Revision, Result: result}}, p.History...)
		p.Current = nil
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

func privateLANURLs(port string) []string {
	// A UDP connection selects the same outbound interface Windows uses for the
	// default route without sending any traffic. This avoids advertising virtual
	// adapter addresses (for example, WSL or VirtualBox) to nearby devices.
	conn, err := net.Dial("udp4", "192.0.2.1:80")
	if err == nil {
		defer conn.Close()
		host, _, splitErr := net.SplitHostPort(conn.LocalAddr().String())
		if ip := net.ParseIP(host); splitErr == nil && ip != nil && ip.IsPrivate() {
			return []string{"http://" + net.JoinHostPort(ip.String(), port)}
		}
	}

	urls := []string{}
	seen := map[string]bool{}
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return urls
	}
	for _, address := range addresses {
		network, ok := address.(*net.IPNet)
		if !ok || network.IP.IsLoopback() {
			continue
		}
		ip := network.IP.To4()
		if ip == nil || !ip.IsPrivate() {
			continue
		}
		url := "http://" + net.JoinHostPort(ip.String(), port)
		if !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}
	return urls
}

func main() {
	lan := flag.Bool("lan", false, "allow connections from devices on your local network")
	port := flag.Int("port", 0, "TCP port to use with --lan (default: 8080)")
	flag.Parse()
	if !*lan && *port != 0 {
		log.Fatal("--port can only be used with --lan")
	}
	if *lan && *port == 0 {
		*port = 8080
	}
	if *lan && (*port < 1 || *port > 65535) {
		log.Fatal("--port must be between 1 and 65535")
	}
	b, err := loadBank()
	if err != nil {
		log.Fatal("Could not load the built-in question bank: ", err)
	}
	unlock, err := acquireServerLock()
	if err != nil {
		log.Fatal("Could not start US Civics Practice: ", err)
	}
	defer unlock()
	listenAddress := "127.0.0.1:0"
	if *lan {
		listenAddress = net.JoinHostPort("0.0.0.0", fmt.Sprint(*port))
	}
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatal("Could not open server: ", err)
	}
	defer listener.Close()
	_, listenerPort, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		log.Fatal("Could not determine server port: ", err)
	}
	localAddress := net.JoinHostPort("127.0.0.1", listenerPort)
	lockPath, _ := serverLockPath()
	if err := os.WriteFile(lockPath, []byte(localAddress), 0600); err != nil {
		log.Fatal("Could not record application startup: ", err)
	}

	st, storeErr := loadStore()
	if storeErr != nil {
		log.Printf("History unavailable: %v", storeErr)
	}
	if st == nil {
		p := newProfile("My Practice")
		st = &dataStore{data: AppData{SchemaVersion: dataSchemaVersion, ActiveProfileID: p.ID, Profiles: []Profile{p}}}
	}
	s := &server{bank: b, sessions: map[string]*Session{}, store: st, storeErr: storeErr}
	if st.active().Current != nil && !st.active().Current.Submitted {
		s.sessions[st.active().Current.ID] = st.active().Current
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", s.api)
	sub, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal("Could not load built-in web assets: ", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	url := "http://" + localAddress
	log.Printf("US Civics Practice is running at %s", url)
	if *lan {
		urls := privateLANURLs(listenerPort)
		if len(urls) == 0 {
			log.Println("Local-network access is enabled, but no private IPv4 address was found. Check your network connection.")
		} else {
			log.Println("Local-network access is enabled. On another device, open one of these addresses:")
			for _, lanURL := range urls {
				log.Println(lanURL)
			}
		}
		log.Println("Allow the Windows Firewall prompt only on Private networks. Do not enable router port forwarding.")
	}
	log.Println("Close this window or press Ctrl+C when you are finished studying.")
	if err := openBrowser(url); err != nil {
		log.Printf("Could not open your browser automatically. Open %s instead: %v", url, err)
	}
	if err := http.Serve(listener, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Local server stopped unexpectedly: ", err)
	}
}
