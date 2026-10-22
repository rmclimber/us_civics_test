package main

import (
	"strings"
	"testing"
)

func sampleBank() QuestionBank {
	return QuestionBank{Questions: []Question{{ID: 1, Question: "One?", Category: "Test", Subcategory: "One", AcceptableAnswers: []string{"Yes", "Yep"}, RequiredAnswerCount: 1, Distractors: []string{"No", "Never", "Maybe"}}, {ID: 2, Question: "Name two?", Category: "Test", Subcategory: "Two", AcceptableAnswers: []string{"A", "B", "C"}, RequiredAnswerCount: 2, Distractors: []string{"D", "E", "F"}}}}
}
func TestSelectionAndStudyOrder(t *testing.T) {
	b := sampleBank()
	qs, e := chooseQuestions(b, "study", 2)
	if e != nil || qs[0].ID != 1 || qs[1].ID != 2 {
		t.Fatal("study order must be deterministic")
	}
	qs, e = chooseQuestions(b, "subset", 2)
	if e != nil || qs[0].ID == qs[1].ID {
		t.Fatal("random selection must not duplicate questions")
	}
}
func TestMultiAnswerGrading(t *testing.T) {
	s, e := makeSession(sampleBank(), "study", "multiple-choice", 2)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range s.Presented {
		ids := []string{}
		for _, o := range p.Options {
			if s.Correct[p.ID][o.ID] {
				ids = append(ids, o.ID)
			}
		}
		if e = updateResponse(s, p.ID, ids); e != nil {
			t.Fatal(e)
		}
	}
	r, e := grade(s)
	if e != nil || r.Correct != 2 {
		t.Fatalf("want perfect result, got %#v %v", r, e)
	}
}
func TestMultiAnswerCanBeSavedIncrementally(t *testing.T) {
	s, e := makeSession(sampleBank(), "study", "multiple-choice", 2)
	if e != nil {
		t.Fatal(e)
	}
	p := s.Presented[1]
	correct := []string{}
	for _, o := range p.Options {
		if s.Correct[p.ID][o.ID] {
			correct = append(correct, o.ID)
		}
	}
	if e = updateResponse(s, p.ID, correct[:1]); e != nil {
		t.Fatalf("first checkbox selection should save: %v", e)
	}
	if len(s.Responses[p.ID]) != 1 {
		t.Fatal("partial selection was not retained")
	}
	if e = updateResponse(s, p.ID, correct); e != nil {
		t.Fatalf("complete selection should save: %v", e)
	}
}
func TestIncompleteAndStandardThreshold(t *testing.T) {
	s, _ := makeSession(sampleBank(), "study", "multiple-choice", 2)
	if _, e := grade(s); e == nil {
		t.Fatal("unanswered submission should fail")
	}
	if !meetsStandard(20, 20, 12) || meetsStandard(20, 20, 11) || meetsStandard(19, 19, 19) {
		t.Fatal("standard threshold must be 12 of a fully scored 20-question quiz")
	}
}
func TestValidation(t *testing.T) {
	b := sampleBank()
	b.Questions = append(b.Questions, make([]Question, 126)...)
	if validateBank(b) == nil {
		t.Fatal("incomplete bank should fail validation")
	}
}
func TestEmbeddedQuestionBank(t *testing.T) {
	b, err := loadBank()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Questions) != 128 {
		t.Fatalf("want 128 questions, got %d", len(b.Questions))
	}
}
func TestOralPracticeRequiresRevealAndSelfGrade(t *testing.T) {
	s, e := makeSession(sampleBank(), "study", "oral", 2)
	if e != nil {
		t.Fatal(e)
	}
	if e = selfGrade(s, 1, true); e == nil {
		t.Fatal("must reveal before self-grading")
	}
	for _, q := range s.Presented {
		if e = reveal(s, q.ID); e != nil {
			t.Fatal(e)
		}
		if e = selfGrade(s, q.ID, true); e != nil {
			t.Fatal(e)
		}
	}
	r, e := grade(s)
	if e != nil || r.Correct != 2 || r.ScoredTotal != 2 {
		t.Fatalf("oral grading failed: %#v %v", r, e)
	}
}
func TestPreferencesDefaults(t *testing.T) {
	p := defaultPreferences()
	if p.Theme != "system" || p.DefaultQuizSize != 20 || p.PracticeMode != "multiple-choice" {
		t.Fatalf("unexpected defaults: %#v", p)
	}
}

func TestDisplayOptionText(t *testing.T) {
	if got := displayOptionText("(Thomas) Jefferson"); got != "Thomas Jefferson" {
		t.Fatalf("unexpected display text: %q", got)
	}
	if got := displayOptionText("Federalist Papers"); got != "Federalist Papers" {
		t.Fatalf("unexpected display text: %q", got)
	}
	if got := displayOptionText("Doubled the size of the United States (Louisiana Purchase)"); got != "Doubled the size of the United States" {
		t.Fatalf("unexpected display text: %q", got)
	}
}

func TestDistractorsUseDisplayText(t *testing.T) {
	bank := QuestionBank{Questions: []Question{{
		ID:                  36,
		Question:            "How many years?",
		Category:            "Test",
		AcceptableAnswers:   []string{"Four (4) years"},
		RequiredAnswerCount: 1,
		Distractors:         []string{"Two (2) years", "Six (6) years", "Ten (10) years"},
	}}}
	s, err := makeSession(bank, "study", "multiple-choice", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range s.Presented[0].Options {
		if option.Text == "Four (4) years" || option.Text == "Two (2) years" || option.Text == "Six (6) years" || option.Text == "Ten (10) years" {
			t.Fatalf("option was not normalized: %q", option.Text)
		}
	}
}

func TestAllMultipleChoiceOptionsUseNormalizedText(t *testing.T) {
	bank, err := loadBank()
	if err != nil {
		t.Fatal(err)
	}
	session, err := makeSession(bank, "study", "multiple-choice", len(bank.Questions))
	if err != nil {
		t.Fatal(err)
	}
	for _, question := range session.Presented {
		for _, option := range question.Options {
			if strings.Contains(option.Text, "(") || strings.Contains(option.Text, ")") {
				t.Fatalf("question %d has an unnormalized option: %q", question.ID, option.Text)
			}
		}
	}
}
