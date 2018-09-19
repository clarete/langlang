all: build

BIN	:= peg.bin pegx.bin
BINX	:= peg.binx pegx.binx abnf.binx csv.binx
PYTHON	?= /usr/bin/python

%.bin:   %.peg; $(PYTHON) ../peg.py -q -c -o $@ -g $<
%.binx: %.pegx; $(PYTHON) ../peg.py -q -c -o $@ -g $<
build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
