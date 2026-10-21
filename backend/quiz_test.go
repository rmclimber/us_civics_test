package main

import "testing"

func sampleBank() QuestionBank { return QuestionBank{Questions:[]Question{{ID:1,Question:"One?",Category:"Test",Subcategory:"One",AcceptableAnswers:[]string{"Yes","Yep"},RequiredAnswerCount:1,Distractors:[]string{"No","Never","Maybe"}},{ID:2,Question:"Name two?",Category:"Test",Subcategory:"Two",AcceptableAnswers:[]string{"A","B","C"},RequiredAnswerCount:2,Distractors:[]string{"D","E","F"}}}} }
func TestSelectionAndStudyOrder(t *testing.T){ b:=sampleBank(); qs,e:=chooseQuestions(b,"study",2);if e!=nil||qs[0].ID!=1||qs[1].ID!=2{t.Fatal("study order must be deterministic")};qs,e=chooseQuestions(b,"subset",2);if e!=nil||qs[0].ID==qs[1].ID{t.Fatal("random selection must not duplicate questions")}}
func TestMultiAnswerGrading(t *testing.T){s,e:=makeSession(sampleBank(),"study",2);if e!=nil{t.Fatal(e)};for _,p:=range s.Presented{ids:=[]string{};for _,o:=range p.Options{if s.Correct[p.ID][o.ID]{ids=append(ids,o.ID)}};if e=updateResponse(s,p.ID,ids);e!=nil{t.Fatal(e)}};r,e:=grade(s);if e!=nil||r.Correct!=2{t.Fatalf("want perfect result, got %#v %v",r,e)}}
func TestIncompleteAndStandardThreshold(t *testing.T){s,_:=makeSession(sampleBank(),"study",2);if _,e:=grade(s);e==nil{t.Fatal("unanswered submission should fail")};if !meetsStandard(20,20,12)||meetsStandard(20,20,11)||meetsStandard(19,19,19){t.Fatal("standard threshold must be 12 of a fully scored 20-question quiz")}}
func TestValidation(t *testing.T){ b:=sampleBank(); b.Questions=append(b.Questions,make([]Question,126)...);if validateBank(b)==nil{t.Fatal("incomplete bank should fail validation")} }
