all: bin

ifeq ("$(origin DEBUG)", "command line")
  OPTFLAGS ?= -g -O0
else
  OPTFLAGS ?= -O1 -flto
endif
ifeq ("$(origin VERBOSE)", "command line")
  OPTFLAGS = $(OPTFLAGS) -DDEBUG
endif

# Configurable
CPPFLAGS ?= $(OPTFLAGS)
CFLAGS   ?= -Wall -Werror -Wpedantic $(DBGFLAGS)

# Source and output files
MATCH	:= match
SRCS	:= peg.c value.c
DEPDIR	:= .d

# Source and output files for test target
TEST	:= test
$(TEST): $(patsubst %.c,%.o,$(SRCS) test.c); $(CC) $(DBGFLAGS) -o $@ $^

# Handle header dependency. Huge thanks to the following article:
# http://make.mad-scientist.net/papers/advanced-auto-dependency-generation/
$(shell mkdir -p $(DEPDIR) >/dev/null)
DEPFLAGS    = -MT $@ -MMD -MP -MF $(DEPDIR)/$*.Td
COMPILE.c   = $(CC) $(DEPFLAGS) $(CFLAGS) $(CPPFLAGS) $(TARGET_ARCH) -c
POSTCOMPILE = @mv -f $(DEPDIR)/$*.Td $(DEPDIR)/$*.d && touch $@

# How to compile each source file taking care of updating dependency
# files.
$(DEPDIR)/%.d:;
.PRECIOUS: $(DEPDIR)/%.d
%.o: %.c
%.o: %.c $(DEPDIR)/%.d
	$(COMPILE.c) $(OUTPUT_OPTION) $<
	$(POSTCOMPILE)

# Generate objects and match binary
bin: $(MATCH)
$(MATCH): $(patsubst %.c,%.o,$(SRCS) match.c); $(CC) -o $@ $^ $(DBGFLAGS)

# Get rid of garbage
clean:; -rm $(MATCH) $(TEST) *.o

# Include the dependency rules for each source file
include $(wildcard $(patsubst %,$(DEPDIR)/%.d,$(basename $(SRCS))))
