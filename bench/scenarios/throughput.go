package scenarios

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func RunThroughput(c Client, n, conc int, outDir string) {
	path := filepath.Join(outDir, "throughput.csv")
	var f *os.File
	var err error
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		f, err = os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		w := csv.NewWriter(f)
		_ = w.Write([]string{"conc", "n", "completed", "errors", "elapsed_s", "rps"})
		w.Flush()
	} else {
		f, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	var done, errs int64
	t0 := time.Now()
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			id, err := c.Create("")
			if err != nil {
				atomic.AddInt64(&errs, 1)
				return
			}
			_ = c.Destroy(id)
			atomic.AddInt64(&done, 1)
		}()
	}
	wg.Wait()
	elapsed := time.Since(t0).Seconds()
	_ = w.Write([]string{
		strconv.Itoa(conc),
		strconv.Itoa(n),
		strconv.FormatInt(done, 10),
		strconv.FormatInt(errs, 10),
		fmt.Sprintf("%.3f", elapsed),
		fmt.Sprintf("%.3f", float64(done)/elapsed),
	})
}
