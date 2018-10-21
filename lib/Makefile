all: build

include ../common.mk

$(call GEN_BIN)
$(call GEN_BINX)

BIN	:= peg.bin pegx.bin
BINX	:= peg.binx pegx.binx abnf.binx csv.binx

build: $(BIN) $(BINX)
clean:; -rm $(BIN) $(BINX)
