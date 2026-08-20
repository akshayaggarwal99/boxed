.PHONY: build coldstart throughput overhead agent escapes plots all clean

ENDPOINT?=http://127.0.0.1:8080
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
	./bin/boxed-bench --scenario=agent --n=20 --endpoint=$(ENDPOINT) --api-key=$(KEY) --out=results

escapes:
	ENDPOINT=$(ENDPOINT) BOXED_API_KEY=$(KEY) bash security/escapes.sh > results/escapes.csv

plots:
	python3 analyze/plots.py results ../paper-v2/figures
	python3 analyze/stats.py results ../paper-v2/tables

all: coldstart throughput overhead agent escapes plots

clean:
	rm -rf bin results/*.csv
