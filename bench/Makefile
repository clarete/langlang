all: bin

include ../common.mk

BINX	:= csv0.binx csv0.nc.binx json0.binx json0.nc.binx

BENCH	:= bench
SRCS	:= bench.c
OBJS	:= $(patsubst %.c,%.o,$(SRCS))

PEGO	:= ../peg.o ../value.o ../io.o
ALLO	:= $(OBJS) $(PEGO)

CSVDATA := ./data/1.a.csv ./data/1.b.csv ./data/1.c.csv

$(call GEN_BINX)
$(call GEN_CC_DEPS,$(SRCS))

$(BENCH): $(ALLO); $(CCC) -o $@ $^
$(CSVDATA): data; python3 gendata.py
data:; mkdir -p $@
bin: $(BENCH) $(BIN) $(BINX)
run: bin $(CSVDATA); ./bench
clean:; -rm $(OBJS) $(BENCH) $(BIN) $(BINX)
