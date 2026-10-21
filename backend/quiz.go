package main

import (
 cryptorand "crypto/rand"
 "encoding/hex"
 "errors"
 mathrand "math/rand/v2"
 "sort"
 "time"
)

type QuestionBank struct { Revision string `json:"revision"`; OfficialSource string `json:"officialSource"`; UpdatesSource string `json:"updatesSource"`; Questions []Question `json:"questions"` }
type Question struct { ID int `json:"id"`; Question string `json:"question"`; Category string `json:"category"`; Subcategory string `json:"subcategory"`; AcceptableAnswers []string `json:"acceptableAnswers"`; RequiredAnswerCount int `json:"requiredAnswerCount"`; Special6520 bool `json:"special6520"`; DynamicAnswer *DynamicAnswer `json:"dynamicAnswer"`; Distractors []string `json:"distractors"` }
type DynamicAnswer struct { Kind string `json:"kind"`; Status string `json:"status"`; VerificationURL string `json:"verificationUrl"`; Note string `json:"note"`; AsOf *string `json:"asOf"` }
type Option struct { ID string `json:"id"`; Text string `json:"text"` }
type PresentedQuestion struct { ID int `json:"id"`; Question string `json:"question"`; Category string `json:"category"`; RequiredAnswerCount int `json:"requiredAnswerCount"`; Dynamic bool `json:"dynamic"`; Options []Option `json:"options"` }
type Session struct { ID string; Questions []Question; Presented []PresentedQuestion; Correct map[int]map[string]bool; Responses map[int][]string; Submitted bool; Created time.Time }
type ResultItem struct { Question PresentedQuestion `json:"question"`; Selected []Option `json:"selected"`; Correct bool `json:"correct"`; AcceptedAnswers []string `json:"acceptedAnswers"`; Dynamic *DynamicAnswer `json:"dynamicAnswer,omitempty"` }
type QuizResult struct { Correct int `json:"correct"`; ScoredTotal int `json:"scoredTotal"`; Total int `json:"total"`; Percentage float64 `json:"percentage"`; PassedStandard bool `json:"passedStandard"`; Items []ResultItem `json:"items"` }

func newID() string { b:=make([]byte,16); _,_=cryptorand.Read(b); return hex.EncodeToString(b) }
func shuffled(in []Question) []Question { out:=append([]Question(nil),in...); mathrand.Shuffle(len(out),func(i,j int){out[i],out[j]=out[j],out[i]}); return out }
func chooseQuestions(bank QuestionBank, mode string, count int) ([]Question,error) { if count<1 || count>len(bank.Questions) { return nil,errors.New("count must be between 1 and 128") }; if mode=="study" { return append([]Question(nil),bank.Questions[:count]...),nil }; if mode!="random" && mode!="subset" { return nil,errors.New("invalid mode") }; return shuffled(bank.Questions)[:count],nil }
func optionID() string { return newID()[:10] }
func makeSession(bank QuestionBank, mode string, count int) (*Session,error) { qs,err:=chooseQuestions(bank,mode,count); if err!=nil{return nil,err}; s:=&Session{ID:newID(),Questions:qs,Correct:map[int]map[string]bool{},Responses:map[int][]string{},Created:time.Now()}; for _,q:=range qs { p:=PresentedQuestion{ID:q.ID,Question:q.Question,Category:q.Category,RequiredAnswerCount:q.RequiredAnswerCount,Dynamic:q.DynamicAnswer!=nil}; answers:=shuffledAnswers(q.AcceptableAnswers,q.RequiredAnswerCount); options:=make([]Option,0,4); correct:=map[string]bool{}; for _,a:=range answers { id:=optionID(); options=append(options,Option{ID:id,Text:a}); correct[id]=true }; for i:=0;len(options)<max(4,len(answers)+3);i++ { options=append(options,Option{ID:optionID(),Text:q.Distractors[i]}) }; mathrand.Shuffle(len(options),func(i,j int){options[i],options[j]=options[j],options[i]}); p.Options=options; s.Presented=append(s.Presented,p); s.Correct[q.ID]=correct }; return s,nil }
func shuffledAnswers(in []string, count int) []string { out:=append([]string(nil),in...); mathrand.Shuffle(len(out),func(i,j int){out[i],out[j]=out[j],out[i]}); return out[:count] }
func min(a,b int) int { if a<b{return a}; return b }
func max(a,b int) int { if a>b{return a}; return b }
func updateResponse(s *Session, qid int, ids []string) error { if s.Submitted{return errors.New("quiz already submitted")}; var p *PresentedQuestion; for i:=range s.Presented {if s.Presented[i].ID==qid {p=&s.Presented[i];break}}; if p==nil{return errors.New("question is not in this quiz")}; if len(ids)!=p.RequiredAnswerCount{return errors.New("select the required number of responses")}; seen:=map[string]bool{}; valid:=map[string]bool{}; for _,o:=range p.Options{valid[o.ID]=true}; for _,id:=range ids{if !valid[id]||seen[id]{return errors.New("invalid answer option")};seen[id]=true}; s.Responses[qid]=append([]string(nil),ids...);return nil }
func grade(s *Session) (QuizResult,error) { if s.Submitted { return QuizResult{},errors.New("quiz already submitted") }; for _,q:=range s.Questions { if q.DynamicAnswer==nil && len(s.Responses[q.ID])!=q.RequiredAnswerCount {return QuizResult{},errors.New("all stable questions must be answered before submitting")} }; result:=QuizResult{Total:len(s.Questions)}; for i,q:=range s.Questions { p:=s.Presented[i]; ids:=s.Responses[q.ID]; selected:=[]Option{}; lookup:=map[string]Option{}; for _,o:=range p.Options{lookup[o.ID]=o}; for _,id:=range ids{selected=append(selected,lookup[id])}; correct:=false; if q.DynamicAnswer==nil { result.ScoredTotal++; correct=len(ids)==len(s.Correct[q.ID]); for _,id:=range ids{if !s.Correct[q.ID][id]{correct=false}}; if correct {result.Correct++} }; result.Items=append(result.Items,ResultItem{Question:p,Selected:selected,Correct:correct,AcceptedAnswers:q.AcceptableAnswers,Dynamic:q.DynamicAnswer}) }; if result.ScoredTotal>0 {result.Percentage=float64(result.Correct)*100/float64(result.ScoredTotal)}; result.PassedStandard=meetsStandard(result.Total,result.ScoredTotal,result.Correct); s.Submitted=true; return result,nil }
func meetsStandard(total, scoredTotal, correct int) bool { return total==20 && scoredTotal==20 && correct>=12 }
func sortQuestions(qs []Question) {sort.Slice(qs,func(i,j int)bool{return qs[i].ID<qs[j].ID})}
