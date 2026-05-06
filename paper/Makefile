PAPER=main

.PHONY: all clean arxiv view

all: $(PAPER).pdf

$(PAPER).pdf: $(PAPER).tex refs.bib tables/numbers.tex
	tectonic -X compile $(PAPER).tex --keep-intermediates --keep-logs

clean:
	rm -f $(PAPER).pdf $(PAPER).aux $(PAPER).bbl $(PAPER).blg $(PAPER).log $(PAPER).out $(PAPER).fls $(PAPER).fdb_latexmk

arxiv: $(PAPER).pdf
	mkdir -p ../dist
	tar czf ../dist/boxed-arxiv.tar.gz \
	  $(PAPER).tex $(PAPER).bbl refs.bib \
	  figures tables

view: $(PAPER).pdf
	open $(PAPER).pdf
