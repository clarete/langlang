all: build

T      ?= -DTEST
bin    := vm
flags  := -O0 -g -pg -Wall -Wpedantic -std=c99 $(T)

vm: vm.o
%.o: %.c debug.h; cc $(flags) -c -o $@ $<
$(bin):; cc $(flags) -o $@ $@.o

build: $(bin)
clean:; -rm $(bin) *.o
