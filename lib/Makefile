all: build

BIN    := peg.bin
PYTHON ?= /usr/bin/python

%.bin: %.peg; $(PYTHON) ../peg.py -q -c -g $<
build: $(BIN)
clean:; -rm $(BIN)
