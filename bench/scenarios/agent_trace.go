package scenarios

// End-to-end agent trace: a model writes a Python function for a HumanEval
// task, Boxed executes it against the official HumanEval test, and the harness
// records where the wall-clock went. Every run records the model identifier,
// request configuration, token usage, stop reason, and the real process exit
// status, and writes the raw completion and the executed program to a JSONL
// file next to the CSV so the pass/fail labels can be audited.
//
// Tasks are the first n entries of the official HumanEval release
// (data/HumanEval.jsonl, Chen et al. 2021), selected in file order.

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

type humanEvalTask struct {
	TaskID     string `json:"task_id"`
	Prompt     string `json:"prompt"`
	EntryPoint string `json:"entry_point"`
	Test       string `json:"test"`
}

func loadHumanEval(path string) ([]humanEvalTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []humanEvalTask
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var t humanEvalTask
		if err := json.Unmarshal(sc.Bytes(), &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, sc.Err()
}

const agentSystemPrompt = "You complete Python functions. The user gives a function signature and docstring. Reply with the complete function, including the signature, inside a single ```python code block. Do not include tests, prints, or explanation."

var codeBlock = regexp.MustCompile("(?s)```(?:python)?\\s*\n(.*?)```")

func extractCode(s string) string {
	m := codeBlock.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return s
}

type agentTraceRecord struct {
	TaskID       string `json:"task_id"`
	Model        string `json:"model"`
	Timestamp    string `json:"timestamp"`
	PromptSHA256 string `json:"prompt_sha256"`
	StopReason   string `json:"stop_reason"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	Completion   string `json:"completion"`
	Program      string `json:"program"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ExitCode     int    `json:"exit_code"`
	Passed       bool   `json:"passed"`
}

func RunAgentTrace(c Client, n int, outDir string) {
	model := os.Getenv("BOXED_AGENT_MODEL")
	if model == "" {
		model = "claude-opus-5"
	}
	dataPath := os.Getenv("HUMANEVAL_PATH")
	if dataPath == "" {
		dataPath = "data/HumanEval.jsonl"
	}
	tasks, err := loadHumanEval(dataPath)
	if err != nil {
		log.Fatalf("load HumanEval: %v", err)
	}
	if n > len(tasks) {
		n = len(tasks)
	}
	client := anthropic.NewClient() // ANTHROPIC_API_KEY from the environment
	ctx := context.Background()

	f, err := os.Create(filepath.Join(outDir, "agent_trace.csv"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"task", "model", "timestamp", "llm_ms", "input_tokens", "output_tokens",
		"stop_reason", "create_ms", "exec_ms", "destroy_ms", "total_ms", "exit_code", "passed"})

	jf, err := os.Create(filepath.Join(outDir, "agent_trace.jsonl"))
	if err != nil {
		log.Fatal(err)
	}
	defer jf.Close()
	jw := json.NewEncoder(jf)

	for i := 0; i < n; i++ {
		t := tasks[i]
		ts := time.Now().UTC().Format(time.RFC3339)
		sum := sha256.Sum256([]byte(agentSystemPrompt + "\x00" + t.Prompt))

		t0 := time.Now()
		resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: agentSystemPrompt}},
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(t.Prompt)),
			},
		})
		llmDur := time.Since(t0)
		if err != nil {
			log.Printf("%s model: %v", t.TaskID, err)
			_ = w.Write([]string{t.TaskID, model, ts, fmt.Sprintf("%.3f", ms(llmDur)), "", "", "error", "", "", "", "", "", "ERR"})
			w.Flush()
			continue
		}
		var reply strings.Builder
		for _, b := range resp.Content {
			if tb, ok := b.AsAny().(anthropic.TextBlock); ok {
				reply.WriteString(tb.Text)
			}
		}
		completion := extractCode(reply.String())

		// prompt supplies the imports; the completion re-declares the function
		// (a second def is legal Python and the later one wins). The official
		// test module defines check(candidate).
		program := "import signal\nsignal.alarm(60)\n" + t.Prompt + "\n" + completion + "\n\n" + t.Test + "\n\ncheck(" + t.EntryPoint + ")\n"

		t1 := time.Now()
		id, err := c.Create("")
		if err != nil {
			log.Printf("%s create: %v", t.TaskID, err)
			continue
		}
		t2 := time.Now()
		res, execErr := c.Exec(id, "python", program)
		t3 := time.Now()
		_ = c.Destroy(id)
		t4 := time.Now()

		code := -1
		if execErr == nil && res.ExitCode != nil {
			code = *res.ExitCode
		}
		passed := execErr == nil && code == 0

		rec := agentTraceRecord{
			TaskID: t.TaskID, Model: model, Timestamp: ts,
			PromptSHA256: hex.EncodeToString(sum[:]),
			StopReason:   string(resp.StopReason),
			InputTokens:  resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens,
			Completion: reply.String(), Program: program,
			Stdout: res.Stdout, Stderr: res.Stderr, ExitCode: code, Passed: passed,
		}
		_ = jw.Encode(rec)

		_ = w.Write([]string{
			t.TaskID, model, ts,
			fmt.Sprintf("%.3f", ms(llmDur)),
			strconv.FormatInt(resp.Usage.InputTokens, 10),
			strconv.FormatInt(resp.Usage.OutputTokens, 10),
			string(resp.StopReason),
			fmt.Sprintf("%.3f", ms(t2.Sub(t1))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t2))),
			fmt.Sprintf("%.3f", ms(t4.Sub(t3))),
			fmt.Sprintf("%.3f", ms(llmDur+t4.Sub(t1))),
			strconv.Itoa(code),
			strconv.FormatBool(passed),
		})
		w.Flush()
	}
}
