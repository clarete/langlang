all: build

BIN	:= sexpr.bin
BINX	:= simpl.binx json.binx
PYTHON	?= $(shell which python)

%.bin:   %.peg; $(PYTHON) ../../peg.py -q -p -c -o $@ -g $<
%.binx: %.pegx; $(PYTHON) ../../peg.py -q -p -c -o $@ -g $<
build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
