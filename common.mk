# Parameters from command line
ifeq ("$(origin DEBUG)", "command line")
  OPTFLAGS ?= -g -O0
else
  OPTFLAGS ?= -O1 -flto
endif
ifeq ("$(origin VERBOSE)", "command line")
  OPTFLAGS := $(OPTFLAGS) -DDEBUG
endif

# Configurable
CPPFLAGS	?= $(OPTFLAGS)
CFLAGS		?= -Wall -Werror -Wpedantic $(DBGFLAGS)
PYTHON		?= $(shell which python)

# Constants
ROOTDIR		:= $(shell git rev-parse --show-toplevel)

# Relative to the current directory
DEPDIR		:= .d
# Handle header dependency. Huge thanks to the following article:
# http://make.mad-scientist.net/papers/advanced-auto-dependency-generation/
DEPFLAGS	= -MT $@ -MMD -MP -MF $(DEPDIR)/$*.Td
CCC		= $(CC) $(DEPFLAGS) $(CFLAGS) $(CPPFLAGS) $(TARGET_ARCH)
COMPILE.c	= $(CCC) -c
POSTCOMPILE	= @mv -f $(DEPDIR)/$*.Td $(DEPDIR)/$*.d && touch $@

define COMPILE_WITH_DEPS
 $(shell mkdir -p $(DEPDIR) >/dev/null)
 $(DEPDIR)/%.d:;
 .PRECIOUS: $(DEPDIR)/%.d
 %.o: %.c
 %.o: %.c $(DEPDIR)/%.d
	$(COMPILE.c) $(OUTPUT_OPTION) $<
	$(POSTCOMPILE)
 include $$(wildcard $(patsubst %,$$(DEPDIR)/%.d,$$(basename $(1))))
endef

