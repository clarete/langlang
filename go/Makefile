parsers := 				\
	tests/basic/basic.ts		\
	tests/import/import.ts		\
	tests/langlang/langlang.ts

all: $(parsers)

clean:; rm $(parsers)

test: all
	npm install --silent
	npm test --silent

tests/basic/basic.ts: tests/basic/basic.peg
	go run ./cmd/langlang -language typescript -grammar $< -output $@

tests/import/import.ts: tests/import/import_gr_expr.peg
	go run ./cmd/langlang -language typescript -grammar $< -output $@

tests/langlang/langlang.ts: ../grammars/langlang.peg
	go run ./cmd/langlang -language typescript -grammar $< -output $@
