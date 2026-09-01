.PHONY: build coldstart throughput overhead agent escapes plots all clean

ENDPOINT?=http://127.0.0.1:8080
RUN?=hardened-2026-09
KEY?=$(BOXED_API_KEY)

build:
	go build -o ./bin/boxed-bench ./cmd/boxed-bench

coldstart: build
	./bin/boxed-bench --scenario=coldstart --n=200 --endpoint=$(ENDPOINT) --api-key=$(KEY) --out=results

throughput: build
	for c in 1 2 4 8 16 32 64; do \
	  ./bin/boxed-bench --scenario=throughput --n=200 --conc=$$c --endpoint=$(ENDPOINT) --api-key=$(KEY) --out=results; \
	done

overhead: build
	./bin/boxed-bench --scenario=overhead --n=30 --endpoint=$(ENDPOINT) --api-key=$(KEY) --out=results

agent: build
	@test -n "$$ANTHROPIC_API_KEY" || (echo "ANTHROPIC_API_KEY not set" && exit 1)
	./bin/boxed-bench --scenario=agent --n=20 --endpoint=$(ENDPOINT) --api-key=$(KEY) --out=results/$(RUN)

escapes:
	ENDPOINT=$(ENDPOINT) BOXED_API_KEY=$(KEY) bash security/escapes.sh > results/escapes.csv

plots:
	python3 analyze/plots.py results/$(RUN) ../paper-v2/figures
	python3 analyze/stats.py results/$(RUN) ../paper-v2/tables results/legacy-2026-05-docker-desktop

all: coldstart throughput overhead agent escapes plots

clean:
	rm -rf bin results/*.csv
