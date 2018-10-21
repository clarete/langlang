all: bin

include $(shell git rev-parse --show-toplevel)/common.mk

# Configurable
OPTFLAGS ?= -g -O0
CPPFLAGS ?= $(OPTFLAGS)
CFLAGS   ?= -Wall -Werror -Wpedantic $(DBGFLAGS)

# Output binaries
CALC := calc
BINX := calc.binx

# Build vars
SRCS := calc.c
OBJS := $(patsubst %.c,%.o,$(SRCS))
PEGO := ../../peg.o ../../value.o
ALLB := $(PEGO) $(OBJS)

$(call GEN_BINX)
$(call GEN_CC_DEPS,$(SRCS))

# Rules for generating objects and binary and for cleaning them up
$(CALC): $(ALLB); $(CC) -o $@ $^ $(DBGFLAGS) -lreadline -lm
bin: $(CALC) $(BINX)
clean:; -rm $(CALC) $(OBJS) $(BINX)
