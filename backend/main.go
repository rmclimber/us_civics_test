package main

import (
 "embed"
 "encoding/json"
 "fmt"
 "io/fs"
 "log"
 "net/http"
 "sync"
 "strings"
)

//go:embed frontend/dist/* data/questions-2025.json
var assets embed.FS

type server struct { bank QuestionBank; sessions map[string]*Session; mu sync.Mutex }
func loadBank() (QuestionBank,error) { var b QuestionBank; raw,err:=assets.ReadFile("data/questions-2025.json"); if err!=nil{return b,err}; err=json.Unmarshal(raw,&b); if err!=nil{return b,err}; return b,validateBank(b) }
func writeJSON(w http.ResponseWriter, code int, v any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(code); _=json.NewEncoder(w).Encode(v) }
func (s *server) api(w http.ResponseWriter,r *http.Request) { w.Header().Set("Cache-Control","no-store"); switch { case r.Method=="GET"&&r.URL.Path=="/api/metadata": writeJSON(w,200,map[string]any{"revision":s.bank.Revision,"total":len(s.bank.Questions)}); return
 case r.Method=="POST"&&r.URL.Path=="/api/quizzes": var in struct{Mode string `json:"mode"`; Count int `json:"count"`}; if json.NewDecoder(r.Body).Decode(&in)!=nil {writeJSON(w,400,map[string]string{"error":"invalid request"});return}; q,err:=makeSession(s.bank,in.Mode,in.Count);if err!=nil{writeJSON(w,400,map[string]string{"error":err.Error()});return};s.mu.Lock();s.sessions[q.ID]=q;s.mu.Unlock();writeJSON(w,201,map[string]any{"id":q.ID,"questions":q.Presented,"responses":q.Responses});return }
 parts:=splitPath(r.URL.Path); if len(parts)<3||parts[1]!="quizzes" {writeJSON(w,404,map[string]string{"error":"not found"});return}; s.mu.Lock(); session:=s.sessions[parts[2]]; s.mu.Unlock(); if session==nil {writeJSON(w,404,map[string]string{"error":"quiz not found"});return}; if r.Method=="GET"&&len(parts)==3 {writeJSON(w,200,map[string]any{"id":session.ID,"questions":session.Presented,"responses":session.Responses,"submitted":session.Submitted});return}; if r.Method=="PUT"&&len(parts)==5&&parts[3]=="responses" {var in struct{OptionIDs []string `json:"optionIds"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{writeJSON(w,400,map[string]string{"error":"invalid request"});return}; var id int;_,_=fmtSscanf(parts[4],&id); if err:=updateResponse(session,id,in.OptionIDs);err!=nil{writeJSON(w,400,map[string]string{"error":err.Error()});return};writeJSON(w,200,map[string]any{"responses":session.Responses});return};if r.Method=="POST"&&len(parts)==4&&parts[3]=="submit" {result,err:=grade(session);if err!=nil{writeJSON(w,400,map[string]string{"error":err.Error()});return};writeJSON(w,200,result);return};writeJSON(w,404,map[string]string{"error":"not found"}) }
func splitPath(p string) []string { out:=[]string{}; for _,x:=range strings.Split(p,"/"){if x!=""{out=append(out,x)}};return out }
func fmtSscanf(v string, out *int) (int,error) { return fmt.Sscanf(v,"%d",out) }
func main() { b,err:=loadBank();if err!=nil{log.Fatal(err)}; s:=&server{bank:b,sessions:map[string]*Session{}}; mux:=http.NewServeMux();mux.HandleFunc("/api/",s.api); sub,_:=fs.Sub(assets,"frontend/dist");mux.Handle("/",http.FileServer(http.FS(sub))); log.Println("US Civics Practice on http://127.0.0.1:8080");log.Fatal(http.ListenAndServe("127.0.0.1:8080",mux)) }
