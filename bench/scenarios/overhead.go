package scenarios

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func RunOverhead(c Client, n int, outDir string) {
	f, err := os.Create(filepath.Join(outDir, "overhead.csv"))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"i", "rss_mib", "cpu_pct"})

	for i := 0; i < n; i++ {
		id, err := c.Create("")
		if err != nil {
			log.Printf("iter %d create: %v", i, err)
			continue
		}
		time.Sleep(5 * time.Second)
		out, err := exec.Command("docker", "stats", "--no-stream",
			"--format", "{{.MemUsage}}|{{.CPUPerc}}", id).Output()
		if err != nil {
			log.Printf("iter %d stats: %v", i, err)
		}
		rss, cpu := parseDockerStats(string(out))
		_ = w.Write([]string{strconv.Itoa(i), fmt.Sprintf("%.2f", rss), fmt.Sprintf("%.2f", cpu)})
		w.Flush()
		_ = c.Destroy(id)
	}
}

// parseDockerStats parses lines like "12.34MiB / 512MiB|0.05%"
func parseDockerStats(s string) (rssMiB, cpu float64) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "|")
	if len(parts) != 2 {
		return 0, 0
	}
	mem := strings.Fields(parts[0])
	if len(mem) == 0 {
		return 0, 0
	}
	rssMiB = parseSize(mem[0])
	cpu, _ = strconv.ParseFloat(strings.TrimSuffix(parts[1], "%"), 64)
	return
}

func parseSize(s string) float64 {
	s = strings.TrimSpace(s)
	suffixes := []struct {
		s string
		f float64
	}{
		{"GiB", 1024}, {"MiB", 1}, {"KiB", 1.0 / 1024}, {"B", 1.0 / (1024 * 1024)},
	}
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf.s) {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, suf.s), 64)
			return v * suf.f
		}
	}
	return 0
}
