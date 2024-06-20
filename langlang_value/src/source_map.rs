#[derive(Clone, Debug, Default, PartialEq, PartialOrd, Eq, Hash)]
pub struct Position {
    /// number of chars have been seen since the begining of the input
    pub offset: usize,
    /// number of EOL sequences seen since the begining of the input
    pub line: usize,
    /// number of chars seen since the begining of the line
    pub column: usize,
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

impl ToString for Position {
    fn to_string(&self) -> String {
        format!("{}:{}", self.line, self.column)
    }
}

#[derive(Clone, Debug, Default, PartialEq, PartialOrd, Eq, Hash)]
pub struct Span {
    pub start: Position,
    pub end: Position,
}

impl Span {
    pub fn new(start: Position, end: Position) -> Self {
        Self { start, end }
    }
}

impl ToString for Span {
    fn to_string(&self) -> String {
        format!("{}-{}", self.start.to_string(), self.end.to_string())
    }
}
