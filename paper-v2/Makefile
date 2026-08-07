# Boxed paper (IEEE conference format).
#   make        -> Boxed-IEEE.pdf
#   make clean
#
# Built with tectonic, which fetches the packages it needs on first run;
# there is no TeX Live prerequisite.   brew install tectonic

PDF := Boxed-IEEE.pdf

$(PDF): main.tex tables/numbers.tex tables/constants.tex tables/comparison.tex refs.bib
	tectonic -X compile main.tex --keep-intermediates
	cp main.pdf $(PDF)

clean:
	rm -f main.aux main.bbl main.blg main.log main.out main.pdf

.PHONY: clean
