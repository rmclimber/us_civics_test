"""Generate the bundled question bank from the official USCIS M-1778 PDF text.

Usage: python scripts/generate_question_bank.py source.txt data/questions-2025.json
The input is extracted from the downloaded official PDF with pypdf.
"""
import json, re, sys
from pathlib import Path

SOURCE = "https://www.uscis.gov/sites/default/files/document/questions-and-answers/2025-Civics-Test-128-Questions-and-Answers.pdf"
UPDATES = "https://www.uscis.gov/citizenship/testupdates"

def clean(text):
    return re.sub(r"\s+", " ", text).strip()

def required_count(question):
    q = question.lower()
    if re.search(r"\b(name|what are) (two|three|five)\b", q):
        return {"two": 2, "three": 3, "five": 5}[re.search(r"\b(name|what are) (two|three|five)\b", q).group(2)]
    return 1

def dynamic_for(qid):
    kinds = {23:"location",29:"location",30:"appointment",38:"election",39:"election",57:"appointment",61:"location",62:"location"}
    if qid not in kinds: return None
    return {"kind": kinds[qid], "status": "unresolved", "asOf": None, "verificationUrl": UPDATES,
            "note": "This answer varies by current officeholder or the applicant's location. It is shown for study but excluded from automatic scoring."}

def main():
    raw = Path(sys.argv[1]).read_text(encoding="utf-8")
    # Strip page furniture and repair OCR line breaks, retaining the bullet lines.
    lines=[]
    for line in raw.splitlines():
        line=line.strip()
        if not line or re.match(r"^\d+ of 19$", line) or line == "uscis.gov/citizenship" or line.startswith("M-1778"): continue
        lines.append(line)
    category=""; subcategory=""; records=[]; current=None; answer_lines=[]
    def finish():
        nonlocal current, answer_lines
        if not current: return
        answers=[clean(a) for a in answer_lines if clean(a)]
        qid=current["id"]
        # Deliberately explicit false choices avoid presenting another potentially valid USCIS answer.
        distractors=[f"Incorrect response {letter}" for letter in ("A","B","C","D","E","F")]
        current.update({"category":category,"subcategory":subcategory,"acceptableAnswers":answers,
                        "requiredAnswerCount":required_count(current["question"]),"special6520":current.pop("star"),
                        "dynamicAnswer":dynamic_for(qid),"distractors":distractors})
        records.append(current); current=None; answer_lines=[]
    i=0
    while i < len(lines):
        line=lines[i]
        if line in ("AMERICAN GOVERNMENT","AMERICAN HISTORY","SYMBOLS AND HOLIDAYS"):
            finish(); category=line.title(); i+=1; continue
        if re.match(r"^[ABC]: ",line):
            finish(); subcategory=line[3:]; i+=1; continue
        match=re.match(r"^(\d{1,3})\.\s*(.*)$",line)
        if match:
            finish(); qid=int(match.group(1)); question=match.group(2); star=False; i+=1
            # Continue wrapping question lines until first bullet/star/category/new question.
            while i<len(lines) and not lines[i].startswith("•") and lines[i] != "*" and not re.match(r"^(\d{1,3})\.\s",lines[i]) and not re.match(r"^[ABC]: ",lines[i]) and lines[i] not in ("AMERICAN GOVERNMENT","AMERICAN HISTORY","SYMBOLS AND HOLIDAYS"):
                question += " " + lines[i]; i+=1
            if i<len(lines) and lines[i]=="*": star=True; i+=1
            current={"id":qid,"question":clean(question),"star":star}; continue
        if line.startswith("•"):
            answer_lines.append(line[1:].strip()); i+=1
            while i<len(lines) and not lines[i].startswith("•") and lines[i] != "*" and not re.match(r"^(\d{1,3})\.\s",lines[i]) and not re.match(r"^[ABC]: ",lines[i]) and lines[i] not in ("AMERICAN GOVERNMENT","AMERICAN HISTORY","SYMBOLS AND HOLIDAYS"):
                answer_lines[-1] += " " + lines[i]; i+=1
            continue
        i+=1
    finish()
    bank={"revision":"M-1778 (09/25)","title":"128 Civics Questions and Answers (2025 version)","officialSource":SOURCE,"updatesSource":UPDATES,"questions":records}
    Path(sys.argv[2]).parent.mkdir(parents=True,exist_ok=True)
    Path(sys.argv[2]).write_text(json.dumps(bank,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
    if len(records)!=128: raise SystemExit(f"Expected 128 records, got {len(records)}")

if __name__=="__main__": main()
