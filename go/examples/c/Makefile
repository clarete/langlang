TARGET  = test

OBJECTS = parser.o main.o

GRAMMAR = ../../../grammars/langlang.peg

all: $(TARGET)

$(TARGET): $(OBJECTS)
	$(CC) $(OBJECTS) -o $@

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

parser.c:
	go run ../../cmd/langlang	\
		-grammar $(GRAMMAR)	\
		-output-language c	\
		-output-path ./parser.c \
		-c-header-path ./parser.h

clean: $(TARGET) $(OBJECTS) parser.c parser.h
	rm -r $^

.PHONY: clean
