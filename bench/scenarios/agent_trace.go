package scenarios

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Tasks: subset of HumanEval-style problems. Cite HumanEval (Chen et al., 2021).
var humanEval = []struct {
	id     string
	prompt string
	check  string // python expression that must eval True after running gen code
}{
	{"sum_list", "Write a python function `sum_list(xs)` returning the sum of a list of ints. Output ONLY the function body in a ```python block.", "sum_list([1,2,3])==6 and sum_list([])==0"},
	{"is_prime", "Write `is_prime(n)` returning True iff n is a prime > 1. Output ONLY the function in a ```python block.", "is_prime(2) and is_prime(7) and not is_prime(1) and not is_prime(9)"},
	{"reverse_str", "Write `reverse_str(s)` returning the reverse of string s. Output ONLY the function in ```python.", "reverse_str('abc')=='cba' and reverse_str('')==''"},
	{"fib", "Write `fib(n)` returning the nth Fibonacci number with fib(0)=0, fib(1)=1. Output ONLY in ```python.", "fib(0)==0 and fib(1)==1 and fib(10)==55"},
	{"gcd", "Write `gcd(a,b)` returning greatest common divisor. Output ONLY in ```python.", "gcd(12,8)==4 and gcd(17,5)==1"},
	{"count_vowels", "Write `count_vowels(s)` counting aeiou case-insensitively. Output ONLY in ```python.", "count_vowels('Hello')==2 and count_vowels('xyz')==0"},
	{"flatten", "Write `flatten(xs)` flattening one level of nested lists. Output ONLY in ```python.", "flatten([[1,2],[3]])==[1,2,3]"},
	{"unique", "Write `unique(xs)` returning xs with duplicates removed, preserving order. Output ONLY in ```python.", "unique([1,2,1,3,2])==[1,2,3]"},
	{"max_diff", "Write `max_diff(xs)` returning max(xs)-min(xs), 0 if empty. Output ONLY in ```python.", "max_diff([1,5,3])==4 and max_diff([])==0"},
	{"is_palindrome", "Write `is_palindrome(s)` ignoring case. Output ONLY in ```python.", "is_palindrome('Racecar') and not is_palindrome('hello')"},
	{"count_words", "Write `count_words(s)` returning number of whitespace-separated tokens. Output ONLY in ```python.", "count_words('a b c')==3 and count_words('  ')==0"},
	{"factorial", "Write `factorial(n)` for n>=0. Output ONLY in ```python.", "factorial(5)==120 and factorial(0)==1"},
	{"second_max", "Write `second_max(xs)` returning the second largest distinct element, or None if <2 distinct. Output ONLY in ```python.", "second_max([1,2,3,3])==2 and second_max([1])==None"},
	{"average", "Write `average(xs)` returning the mean as float, 0.0 if empty. Output ONLY in ```python.", "abs(average([1,2,3])-2.0)<1e-9"},
	{"rotate", "Write `rotate(xs,k)` rotating list left by k positions. Output ONLY in ```python.", "rotate([1,2,3,4],1)==[2,3,4,1]"},
	{"sort_desc", "Write `sort_desc(xs)` returning xs sorted descending. Output ONLY in ```python.", "sort_desc([3,1,2])==[3,2,1]"},
	{"capitalize_words", "Write `capitalize_words(s)` capitalizing first letter of each whitespace-separated word. Output ONLY in ```python.", "capitalize_words('hello world')=='Hello World'"},
	{"is_anagram", "Write `is_anagram(a,b)` case-insensitively. Output ONLY in ```python.", "is_anagram('listen','Silent') and not is_anagram('a','b')"},
	{"to_binary", "Write `to_binary(n)` returning the binary string of non-negative int n without prefix. Output ONLY in ```python.", "to_binary(5)=='101' and to_binary(0)=='0'"},
	{"merge_sorted", "Write `merge_sorted(a,b)` merging two sorted lists. Output ONLY in ```python.", "merge_sorted([1,3,5],[2,4])==[1,2,3,4,5]"},
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func callGemini(apiKey, prompt string) (string, time.Duration, error) {
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	body, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{{
			"parts": []map[string]string{{"text": prompt}},
		}},
	})
	t0 := time.Now()
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", time.Since(t0), fmt.Errorf("gemini %d: %s", resp.StatusCode, string(b))
	}
	var g geminiResp
	if err := json.Unmarshal(b, &g); err != nil {
		return "", time.Since(t0), err
	}
	if len(g.Candidates) == 0 || len(g.Candidates[0].Content.Parts) == 0 {
		return "", time.Since(t0), fmt.Errorf("empty completion")
	}
	return g.Candidates[0].Content.Parts[0].Text, time.Since(t0), nil
}

var codeBlock = regexp.MustCompile("(?s)```python\\s*(.*?)\\s*```")

func extractCode(s string) string {
	m := codeBlock.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	// fallback: assume the whole reply is code
	return s
}

func RunAgentTrace(c Client, n int, outDir string) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY required")
	}
	if n > len(humanEval) {
		n = len(humanEval)
	}

	f, err := os.Create(filepath.Join(outDir, "agent_trace.csv"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"task", "llm_ms", "create_ms", "exec_ms", "destroy_ms", "total_ms", "passed"})

	for i := 0; i < n; i++ {
		t := humanEval[i]
		reply, llmDur, err := callGemini(apiKey, t.prompt)
		if err != nil {
			log.Printf("%s gemini: %v", t.id, err)
			_ = w.Write([]string{t.id, fmt.Sprintf("%.3f", ms(llmDur)), "", "", "", "", "ERR"})
			w.Flush()
			continue
		}
		code := extractCode(reply)
		// Build a self-checking program
		fullCode := code + "\nimport sys\nsys.exit(0 if (" + t.check + ") else 1)\n"

		t0 := time.Now()
		id, err := c.Create("")
		if err != nil {
			log.Printf("%s create: %v", t.id, err)
			continue
		}
		t1 := time.Now()
		res, execErr := c.Exec(id, "python", fullCode)
		t2 := time.Now()
		_ = c.Destroy(id)
		t3 := time.Now()

		passed := "false"
		if execErr == nil && res.ExitCode != nil && *res.ExitCode == 0 {
			passed = "true"
		}
		_ = w.Write([]string{
			t.id,
			fmt.Sprintf("%.3f", ms(llmDur)),
			fmt.Sprintf("%.3f", ms(t1.Sub(t0))),
			fmt.Sprintf("%.3f", ms(t2.Sub(t1))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t2))),
			fmt.Sprintf("%.3f", ms(llmDur+t3.Sub(t0))),
			passed,
		})
		w.Flush()
	}
	// avoid unused-import lint when struct fields evolve
	_ = strings.TrimSpace
	_ = strconv.Itoa
}
