package main

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	mathrand "math/rand/v2"
	"sort"
	"strings"
	"time"
)

type QuestionBank struct {
	Revision       string     `json:"revision"`
	OfficialSource string     `json:"officialSource"`
	UpdatesSource  string     `json:"updatesSource"`
	Questions      []Question `json:"questions"`
}
type Question struct {
	ID                  int            `json:"id"`
	Question            string         `json:"question"`
	Category            string         `json:"category"`
	Subcategory         string         `json:"subcategory"`
	AcceptableAnswers   []string       `json:"acceptableAnswers"`
	RequiredAnswerCount int            `json:"requiredAnswerCount"`
	Special6520         bool           `json:"special6520"`
	DynamicAnswer       *DynamicAnswer `json:"dynamicAnswer"`
	Distractors         []string       `json:"distractors"`
}
type DynamicAnswer struct {
	Kind            string  `json:"kind"`
	Status          string  `json:"status"`
	VerificationURL string  `json:"verificationUrl"`
	Note            string  `json:"note"`
	AsOf            *string `json:"asOf"`
}
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type PresentedQuestion struct {
	ID                  int      `json:"id"`
	Question            string   `json:"question"`
	Category            string   `json:"category"`
	RequiredAnswerCount int      `json:"requiredAnswerCount"`
	Dynamic             bool     `json:"dynamic"`
	Options             []Option `json:"options,omitempty"`
}
type Session struct {
	ID         string                  `json:"id"`
	ProfileID  string                  `json:"profileId"`
	Mode       string                  `json:"mode"`
	Order      string                  `json:"order"`
	Questions  []Question              `json:"questions"`
	Presented  []PresentedQuestion     `json:"presented"`
	Correct    map[int]map[string]bool `json:"correct"`
	Responses  map[int][]string        `json:"responses"`
	SelfGrades map[int]bool            `json:"selfGrades"`
	Revealed   map[int]bool            `json:"revealed"`
	ResolvedDynamic map[int]bool        `json:"resolvedDynamic,omitempty"`
	Submitted  bool                    `json:"submitted"`
	Created    time.Time               `json:"created"`
}
type ResultItem struct {
	Question            PresentedQuestion `json:"question"`
	Selected            []Option          `json:"selected"`
	Correct             bool              `json:"correct"`
	AcceptedAnswers     []string          `json:"acceptedAnswers"`
	RequiredAnswerCount int               `json:"requiredAnswerCount"`
	Dynamic             *DynamicAnswer    `json:"dynamicAnswer,omitempty"`
	SelfGraded          bool              `json:"selfGraded"`
}
type QuizResult struct {
	Correct        int          `json:"correct"`
	ScoredTotal    int          `json:"scoredTotal"`
	Total          int          `json:"total"`
	Percentage     float64      `json:"percentage"`
	PassedStandard bool         `json:"passedStandard"`
	Items          []ResultItem `json:"items"`
}

func newID() string { b := make([]byte, 16); _, _ = cryptorand.Read(b); return hex.EncodeToString(b) }
func shuffled(in []Question) []Question {
	out := append([]Question(nil), in...)
	mathrand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
func chooseQuestions(bank QuestionBank, mode string, count int) ([]Question, error) {
	if count < 1 || count > len(bank.Questions) {
		return nil, errors.New("count must be between 1 and 128")
	}
	if mode == "study" {
		return append([]Question(nil), bank.Questions[:count]...), nil
	}
	if mode != "random" && mode != "subset" {
		return nil, errors.New("invalid order")
	}
	return shuffled(bank.Questions)[:count], nil
}
func optionID() string { return newID()[:10] }
func displayOptionText(answer string) string {
	if strings.HasPrefix(answer, "(") {
		if end := strings.Index(answer, ") "); end > 1 {
			answer = answer[1:end] + " " + answer[end+2:]
		}
	}
	var text strings.Builder
	depth := 0
	for _, r := range answer {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				text.WriteRune(r)
			}
		}
	}
	return strings.Join(strings.Fields(text.String()), " ")
}
func makeSession(bank QuestionBank, order, practiceMode string, count int) (*Session, error) {
	if practiceMode != "multiple-choice" && practiceMode != "oral" {
		return nil, errors.New("invalid practice mode")
	}
	qs, err := chooseQuestions(bank, order, count)
	if err != nil {
		return nil, err
	}
	return makeSessionQuestions(qs, order, practiceMode, nil)
}
func makeProfileSession(bank QuestionBank, order, practiceMode string, count int, p *Profile) (*Session,error) {
	eligible:=[]Question{}
	resolved:=map[int][]string{}
	for _,q:=range bank.Questions { if q.DynamicAnswer==nil { eligible=append(eligible,q);continue }; r,ok:=p.CurrentAnswers[q.ID];if ok&&r.Status=="Manually confirmed"&&r.VerifiedAt!=nil&&time.Since(*r.VerifiedAt)<=30*24*time.Hour&&len(r.Answers)>0 {q.AcceptableAnswers=append([]string(nil),r.Answers...);eligible=append(eligible,q);resolved[q.ID]=q.AcceptableAnswers} }
	if count>len(eligible){count=len(eligible)};if count<1{return nil,errors.New("no questions are available")}; var qs []Question;if order=="study"{qs=eligible[:count]}else if order=="random"||order=="subset"{qs=shuffled(eligible)[:count]}else{return nil,errors.New("invalid order")};return makeSessionQuestions(qs,order,practiceMode,resolved)
}
func makeSessionQuestions(qs []Question, order, practiceMode string, resolved map[int][]string) (*Session,error) {
	s := &Session{ID: newID(), Mode: practiceMode, Order: order, Questions: qs, Correct: map[int]map[string]bool{}, Responses: map[int][]string{}, SelfGrades: map[int]bool{}, Revealed: map[int]bool{}, ResolvedDynamic:map[int]bool{}, Created: time.Now()}
	for _, q := range qs {
		if _,ok:=resolved[q.ID];ok{s.ResolvedDynamic[q.ID]=true}
		p := PresentedQuestion{ID: q.ID, Question: q.Question, Category: q.Category, RequiredAnswerCount: q.RequiredAnswerCount, Dynamic: q.DynamicAnswer != nil}
		if practiceMode == "multiple-choice" {
			answers := shuffledAnswers(q.AcceptableAnswers, q.RequiredAnswerCount)
			options := make([]Option, 0, 4)
			correct := map[string]bool{}
			for _, a := range answers {
				id := optionID()
				options = append(options, Option{ID: id, Text: displayOptionText(a)})
				correct[id] = true
			}
			for i := 0; len(options) < max(4, len(answers)+3); i++ {
				options = append(options, Option{ID: optionID(), Text: q.Distractors[i]})
			}
			mathrand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })
			p.Options = options
			s.Correct[q.ID] = correct
		}
		s.Presented = append(s.Presented, p)
	}
	return s, nil
}
func shuffledAnswers(in []string, count int) []string {
	out := append([]string(nil), in...)
	mathrand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out[:count]
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func updateResponse(s *Session, qid int, ids []string) error {
	if s.Submitted {
		return errors.New("quiz already submitted")
	}
	if s.Mode != "multiple-choice" {
		return errors.New("answers are self-graded in oral practice")
	}
	var p *PresentedQuestion
	for i := range s.Presented {
		if s.Presented[i].ID == qid {
			p = &s.Presented[i]
			break
		}
	}
	if p == nil {
		return errors.New("question is not in this quiz")
	}
	if len(ids) > p.RequiredAnswerCount {
		return errors.New("too many responses selected")
	}
	seen := map[string]bool{}
	valid := map[string]bool{}
	for _, o := range p.Options {
		valid[o.ID] = true
	}
	for _, id := range ids {
		if !valid[id] || seen[id] {
			return errors.New("invalid answer option")
		}
		seen[id] = true
	}
	s.Responses[qid] = append([]string(nil), ids...)
	return nil
}
func selfGrade(s *Session, qid int, correct bool) error {
	if s.Submitted {
		return errors.New("quiz already submitted")
	}
	if s.Mode != "oral" {
		return errors.New("only oral practice is self-graded")
	}
	if !s.Revealed[qid] {
		return errors.New("reveal the official answer before grading")
	}
	found := false
	for _, p := range s.Presented {
		if p.ID == qid {
			found = true
		}
	}
	if !found {
		return errors.New("question is not in this quiz")
	}
	s.SelfGrades[qid] = correct
	return nil
}
func reveal(s *Session, qid int) error {
	if s.Mode != "oral" {
		return errors.New("only oral practice has answers to reveal")
	}
	for _, p := range s.Presented {
		if p.ID == qid {
			s.Revealed[qid] = true
			return nil
		}
	}
	return errors.New("question is not in this quiz")
}
func grade(s *Session) (QuizResult, error) {
	if s.Submitted {
		return QuizResult{}, errors.New("quiz already submitted")
	}
	for _, q := range s.Questions {
		if s.Mode == "oral" {
			if _, ok := s.SelfGrades[q.ID]; !ok {
				return QuizResult{}, errors.New("self-grade every question before finishing")
			}
		} else if (q.DynamicAnswer == nil || s.ResolvedDynamic[q.ID]) && len(s.Responses[q.ID]) != q.RequiredAnswerCount {
			return QuizResult{}, errors.New("all stable questions must be answered before submitting")
		}
	}
	result := QuizResult{Total: len(s.Questions)}
	for i, q := range s.Questions {
		p := s.Presented[i]
		ids := s.Responses[q.ID]
		selected := []Option{}
		lookup := map[string]Option{}
		for _, o := range p.Options {
			lookup[o.ID] = o
		}
		for _, id := range ids {
			selected = append(selected, lookup[id])
		}
		correct := false
		if s.Mode == "oral" {
			correct = s.SelfGrades[q.ID]
			result.ScoredTotal++
			if correct {
				result.Correct++
			}
		} else if q.DynamicAnswer == nil || s.ResolvedDynamic[q.ID] {
			result.ScoredTotal++
			correct = len(ids) == len(s.Correct[q.ID])
			for _, id := range ids {
				if !s.Correct[q.ID][id] {
					correct = false
				}
			}
			if correct {
				result.Correct++
			}
		}
		result.Items = append(result.Items, ResultItem{Question: p, Selected: selected, Correct: correct, AcceptedAnswers: q.AcceptableAnswers, RequiredAnswerCount: q.RequiredAnswerCount, Dynamic: q.DynamicAnswer, SelfGraded: s.Mode == "oral"})
	}
	if result.ScoredTotal > 0 {
		result.Percentage = float64(result.Correct) * 100 / float64(result.ScoredTotal)
	}
	result.PassedStandard = meetsStandard(result.Total, result.ScoredTotal, result.Correct)
	s.Submitted = true
	return result, nil
}
func meetsStandard(total, scoredTotal, correct int) bool {
	return total == 20 && scoredTotal == 20 && correct >= 12
}
func sortQuestions(qs []Question) { sort.Slice(qs, func(i, j int) bool { return qs[i].ID < qs[j].ID }) }
