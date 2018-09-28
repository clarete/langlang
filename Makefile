all: bin

SUBDIRS = lib

include common.mk

MATCH	:= match
TEST	:= test
SRCS	:= peg.c value.c
MOBJS	:= $(patsubst %.c,%.o,$(SRCS) match.c)
TOBJS	:= $(patsubst %.c,%.o,$(SRCS) test.c)

bin: $(MATCH) $(SUBDIRS)
clean:; -rm $(MATCH) $(TEST) *.o
$(MATCH): $(MOBJS); $(CCC) -o $@ $^
$(TEST): $(TOBJS); $(CCC) -o $@ $^
$(call $(COMPILE_WITH_DEPS),$(SRCS))

$(SUBDIRS):; $(MAKE) -C $@
.PHONY: $(SUBDIRS)
