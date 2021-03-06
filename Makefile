all: bin

SUBDIRS = lib bench examples/calc examples/tiny

include common.mk

MATCH	:= match
TEST	:= test
SRCS	:= peg.c value.c io.c
MOBJS	:= $(patsubst %.c,%.o,$(SRCS) match.c)
TOBJS	:= $(patsubst %.c,%.o,$(SRCS) test.c)

$(call GEN_CC_DEPS,$(SRCS))

bin: $(MATCH) $(SUBDIRS)
clean: $(SUBDIRS); -rm $(MATCH) $(TEST) $(TOBJS) $(MOBJS)
$(MATCH): $(MOBJS); $(CCC) -o $@ $^
$(TEST): $(TOBJS); $(CCC) -o $@ $^

$(SUBDIRS):; $(MAKE) -C $@ $(MAKECMDGOALS)
.PHONY: $(SUBDIRS)
