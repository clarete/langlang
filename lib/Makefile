all: build

include $(shell git rev-parse --show-toplevel)/common.mk

BIN	:= peg.bin pegx.bin
BINX	:= peg.binx pegx.binx abnf.binx csv.binx

%.bin:   %.peg; $(PYTHON) ../peg.py -q -c -o $@ -g $<
%.binx: %.pegx; $(PYTHON) ../peg.py -q -c -o $@ -g $<
build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
