module github.com/clarete/langlang/benchmarks

go 1.24.3

require (
	github.com/buger/jsonparser v1.1.1
	github.com/clarete/langlang/go v0.0.0-00010101000000-000000000000
	github.com/mna/pigeon v1.3.1-0.20250611183742-2aff2dbdef71
	github.com/pointlander/peg v1.0.2-0.20250528183811-1938d26d48e4
)

require (
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/tree-sitter/go-tree-sitter v0.25.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
)

replace github.com/clarete/langlang/go => ../go
