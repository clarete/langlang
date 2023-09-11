#[derive(Clone, Debug, PartialEq)]
pub struct Position {
    /// number of chars have been seen since the begining of the input
    offset: usize,
    /// number of EOL sequences seen since the begining of the input
    line: usize,
    /// number of chars seen since the begining of the line
    column: usize,
}

impl Position {
    pub fn new(offset: usize, line: usize, column: usize) -> Self {
        Position {
            offset,
            line,
            column,
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Span {
    start: Position,
    end: Position,
}

impl Span {
    pub fn new(start: Position, end: Position) -> Self {
        Self { start, end }
    }
}
