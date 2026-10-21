package main

import ("encoding/json";"errors";"fmt";"os";"path/filepath";"sync";"time")

const dataSchemaVersion = 1
type Preferences struct { Theme string `json:"theme"`; DefaultQuizSize int `json:"defaultQuizSize"`; PracticeMode string `json:"practiceMode"`; State string `json:"state"` }
type Attempt struct { ID string `json:"id"`; CompletedAt time.Time `json:"completedAt"`; Mode string `json:"mode"`; QuestionCount int `json:"questionCount"`; QuestionOrder []int `json:"questionOrder"`; BankRevision string `json:"bankRevision"`; Result QuizResult `json:"result"` }
type AppData struct { SchemaVersion int `json:"schemaVersion"`; Preferences Preferences `json:"preferences"`; Current *Session `json:"current,omitempty"`; History []Attempt `json:"history"` }
type dataStore struct { path string; mu sync.Mutex; data AppData }
func defaultPreferences() Preferences{return Preferences{Theme:"system",DefaultQuizSize:20,PracticeMode:"multiple-choice"}}
func userDataPath() (string,error) { dir,err:=os.UserConfigDir();if err!=nil{return "",err};return filepath.Join(dir,"USCivicsPractice","history.json"),nil }
func loadStore() (*dataStore,error) { path,err:=userDataPath();if err!=nil{return nil,err}; s:=&dataStore{path:path,data:AppData{SchemaVersion:dataSchemaVersion,Preferences:defaultPreferences(),History:[]Attempt{}}}; raw,err:=os.ReadFile(path);if errors.Is(err,os.ErrNotExist){return s,nil};if err!=nil{return nil,err};if err=json.Unmarshal(raw,&s.data);err!=nil{return nil,fmt.Errorf("could not read existing history at %s: %w",path,err)};if s.data.SchemaVersion!=dataSchemaVersion{return nil,fmt.Errorf("history at %s uses incompatible schema version %d",path,s.data.SchemaVersion)};if s.data.Preferences.Theme==""{s.data.Preferences=defaultPreferences()};return s,nil }
func (s *dataStore) save() error { s.mu.Lock();defer s.mu.Unlock();return s.saveLocked() }
func (s *dataStore) saveLocked() error { if err:=os.MkdirAll(filepath.Dir(s.path),0700);err!=nil{return err}; raw,err:=json.MarshalIndent(s.data,"","  ");if err!=nil{return err}; tmp:=s.path+".tmp";if err=os.WriteFile(tmp,raw,0600);err!=nil{return err};if err=os.Chmod(tmp,0600);err!=nil{return err};return os.Rename(tmp,s.path) }
