all: build

include ../common.mk

BIN	:= peg.bin pegx.bin
BINX	:= peg.binx pegx.binx abnf.binx csv.binx

%.bin:   %.peg; $(PEGC) -q -c -o $@ -g $<
%.binx: %.pegx; $(PEGC) -q -c -o $@ -g $<
build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
