use serde::{de, ser};
use std::fmt::{self, Display};

pub type Result<T> = std::result::Result<T, Error>;

#[derive(Debug)]
pub enum Error {
    Message(String),
    IndexError,
    ExpectedChr,
    ExpectedStr,
    ExpectedI64,
    ExpectedBool,
    ExpectedNode,
}

impl std::error::Error for Error {}

impl ser::Error for Error {
    fn custom<T: Display>(msg: T) -> Self {
        Error::Message(msg.to_string())
    }
}

impl de::Error for Error {
    fn custom<T: Display>(msg: T) -> Self {
        Error::Message(msg.to_string())
    }
}

impl Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        match self {
            Error::Message(msg) => write!(f, "{}", msg),
            Error::IndexError => write!(f, "Index Error"),
            Error::ExpectedChr => write!(f, "Expected Chr"),
            Error::ExpectedStr => write!(f, "Expected Str"),
            Error::ExpectedI64 => write!(f, "Expected I64"),
            Error::ExpectedBool => write!(f, "Expected Bool"),
            Error::ExpectedNode => write!(f, "Expected Node"),
        }
    }
}
