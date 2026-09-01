# Boxed paper (IEEE conference format, target IC2E).
#   make            -> regenerate tables + figures from bench/results/$(RUN), build Boxed-IEEE.pdf
#   make pdf        -> build only
#   make clean
# Built with tectonic (brew install tectonic); no TeX Live prerequisite.

RUN ?= hardened-2026-09
export RELEASE_TAG ?= v0.2.0-paper
PDF := Boxed-IEEE.pdf
TECTONIC ?= $(shell command -v tectonic || echo /opt/homebrew/bin/tectonic)
PYTHON ?= python3
BENCH := ../bench

all: tables pdf

tables:
	$(PYTHON) $(BENCH)/analyze/stats.py $(BENCH)/results/$(RUN) tables $(BENCH)/results/legacy-2026-05-docker-desktop
	$(PYTHON) $(BENCH)/analyze/plots.py $(BENCH)/results/$(RUN) figures
	$(BENCH)/analyze/constants.sh tables/constants.tex

pdf: main.tex tables/numbers.tex tables/constants.tex tables/comparison.tex refs.bib
	$(TECTONIC) -X compile main.tex --keep-intermediates
	cp main.pdf $(PDF)

clean:
	rm -f main.aux main.bbl main.blg main.log main.out main.pdf

.PHONY: all tables pdf clean
