all: build

BIN	:= peg.bin
BINX	:= peg.binx abnf.binx csv.binx
PYTHON	?= /usr/bin/python

%.bin:   %.peg; $(PYTHON) ../peg.py -q -c -o $@ -g $<
%.binx: %.pegx; $(PYTHON) ../peg.py -q -c -o $@ -g $<
build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
