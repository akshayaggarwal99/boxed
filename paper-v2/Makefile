# Boxed paper (IEEE conference format, target IC2E).
#   make            -> regenerate tables + figures from bench/results/$(RUN), build Boxed-IEEE.pdf
#   make pdf        -> build only
#   make clean
# Built with tectonic (brew install tectonic); no TeX Live prerequisite.

RUN ?= hardened-2026-09
export RELEASE_TAG ?= v0.3.0-paper
PDF := Boxed-IEEE.pdf
TECTONIC ?= $(shell command -v tectonic || echo /opt/homebrew/bin/tectonic)
PYTHON ?= python3
BENCH := ../bench

all: tables pdf

LINUX ?= linux-2026-09
tables:
	$(PYTHON) $(BENCH)/analyze/stats.py $(BENCH)/results/$(RUN) tables $(BENCH)/results/legacy-2026-05-docker-desktop
	$(PYTHON) $(BENCH)/analyze/stats.py $(BENCH)/results/$(LINUX) tables --prefix Linux
	$(PYTHON) $(BENCH)/analyze/lifecycle_combined.py $(BENCH)/results tables "MacBook Pro M1 Pro, colima VM, laptop in use=$(RUN)" "GCE n2-standard-4, native Linux, idle=$(LINUX)"
	$(PYTHON) $(BENCH)/analyze/runtimes.py $(BENCH)/results tables figures linux-runc-2026-09 linux-runsc-2026-09 linux-kata-2026-09
	$(PYTHON) $(BENCH)/analyze/scaling.py $(BENCH)/results tables figures $(LINUX):4 linux-8core-2026-09:8 linux-16core-2026-09:16
	$(PYTHON) $(BENCH)/analyze/plots.py $(BENCH)/results/$(RUN) figures
	$(BENCH)/analyze/constants.sh tables/constants.tex

pdf: main.tex tables/numbers.tex tables/constants.tex tables/comparison.tex refs.bib
	$(TECTONIC) -X compile main.tex --keep-intermediates
	cp main.pdf $(PDF)

clean:
	rm -f main.aux main.bbl main.blg main.log main.out main.pdf

.PHONY: all tables pdf clean
