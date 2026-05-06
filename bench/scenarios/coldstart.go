package scenarios

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func RunColdStart(c Client, n int, outDir string) {
	f, err := os.Create(filepath.Join(outDir, "coldstart.csv"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"i", "create_ms", "first_exec_ms", "destroy_ms", "total_ms"})

	for i := 0; i < n; i++ {
		t0 := time.Now()
		id, err := c.Create("")
		if err != nil {
			log.Printf("iter %d create: %v", i, err)
			continue
		}
		t1 := time.Now()
		if _, err := c.Exec(id, "python", "print('ok')"); err != nil {
			log.Printf("iter %d exec: %v", i, err)
		}
		t2 := time.Now()
		if err := c.Destroy(id); err != nil {
			log.Printf("iter %d destroy: %v", i, err)
		}
		t3 := time.Now()
		_ = w.Write([]string{
			strconv.Itoa(i),
			fmt.Sprintf("%.3f", ms(t1.Sub(t0))),
			fmt.Sprintf("%.3f", ms(t2.Sub(t1))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t2))),
			fmt.Sprintf("%.3f", ms(t3.Sub(t0))),
		})
		w.Flush()
	}
}
