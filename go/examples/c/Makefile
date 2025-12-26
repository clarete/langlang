TARGET  = test

OBJECTS = parser.o main.o

GRAMMAR = ../../../grammars/langlang.peg

CFLAGS = -g

all: $(TARGET)

$(TARGET): $(OBJECTS)
	$(CC) -g $(OBJECTS) -o $@

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

parser.c:
	go run ../../cmd/langlang	\
		-grammar $(GRAMMAR)	\
		-output-language c	\
		-output-path ./parser.c

clean:
	rm -f $(TARGET) $(OBJECTS) parser.c parser.h

.PHONY: clean
