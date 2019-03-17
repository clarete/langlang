all: bin

include $(shell git rev-parse --show-toplevel)/common.mk

# Configurable
OPTFLAGS ?= -g -O0
CPPFLAGS ?= $(OPTFLAGS)
CFLAGS   ?= -Wall -Werror -Wpedantic $(DBGFLAGS)

# Output binaries
TINY := tiny
BINX := tiny.binx

# Build vars
SRCS := tiny.c
OBJS := $(patsubst %.c,%.o,$(SRCS))
PEGO := ../../peg.o ../../value.o ../../io.o
ALLB := $(PEGO) $(OBJS)

$(call GEN_BINX)
$(call GEN_CC_DEPS,$(SRCS))

# Rules for generating objects and binary and for cleaning them up
$(TINY): $(ALLB); $(CC) -o $@ $^ $(DBGFLAGS) -lreadline -lm
bin: $(TINY) $(BINX)
clean:; -rm $(TINY) $(OBJS) $(BINX)
