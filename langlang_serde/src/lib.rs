mod error;

use langlang_lib::vm;
use serde::de::{self, DeserializeSeed, MapAccess, Visitor};
use serde::Deserialize;

use error::{Error, Result};

pub struct Deserializer<'de> {
    stack: Vec<Vec<&'de vm::Value>>,
}

impl<'de> Deserializer<'de> {
    fn from_val(input: &'de vm::Value) -> Self {
        Self {
            stack: vec![vec![input]],
        }
    }
}

pub fn from_val<'a, T>(input: &'a vm::Value) -> Result<T>
where
    T: Deserialize<'a>,
{
    let mut deserializer = Deserializer::from_val(input);
    let t = T::deserialize(&mut deserializer)?;
    Ok(t)
}

impl<'de> Deserializer<'de> {
    fn enter_node(&mut self, items: &'de [vm::Value]) {
        self.stack.push(items.iter().rev().collect());
    }

    fn leave_node(&mut self) {
        self.stack.pop();
    }

    fn current(&mut self) -> Option<&'de vm::Value> {
        let topframe = &self.stack[self.stack.len() - 1];
        let len = topframe.len();
        if len > 0 {
            return Some(topframe[len - 1]);
        }
        None
    }
}

impl<'de, 'a> de::Deserializer<'de> for &'a mut Deserializer<'de> {
    type Error = Error;

    fn deserialize_any<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self
            .current()
            .ok_or_else(|| Error::Message("Empty".to_string()))?
        {
            vm::Value::Char(_) => self.deserialize_char(visitor),
            vm::Value::String(_) => self.deserialize_str(visitor),
            vm::Value::I64(_) => self.deserialize_i64(visitor),
            vm::Value::Bool(_) => self.deserialize_bool(visitor),
            vm::Value::Node { name, .. } => visitor.visit_borrowed_str(name),
            _ => {
                unimplemented!()
            }
        }
    }

    fn deserialize_bool<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self.current() {
            Some(vm::Value::Bool(v)) => visitor.visit_bool(*v),
            _ => Err(Error::ExpectedBool),
        }
    }

    fn deserialize_i8<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_i16<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_i32<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_i64<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self.current() {
            Some(vm::Value::I64(v)) => visitor.visit_i64(*v),
            _ => Err(Error::ExpectedI64),
        }
    }

    fn deserialize_u8<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_u16<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_u32<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_u64<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_f32<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_f64<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_char<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self.current() {
            Some(vm::Value::Char(c)) => visitor.visit_char(*c),
            _ => Err(Error::ExpectedChr),
        }
    }

    fn deserialize_str<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self.current() {
            Some(vm::Value::String(s)) => visitor.visit_borrowed_str(s),
            _ => Err(Error::ExpectedStr),
        }
    }

    fn deserialize_string<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_str(visitor)
    }

    fn deserialize_bytes<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_byte_buf<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_option<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_unit<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_unit_struct<V>(self, _name: &'static str, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_unit(visitor)
    }

    fn deserialize_newtype_struct<V>(self, _name: &'static str, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        visitor.visit_newtype_struct(self)
    }

    fn deserialize_seq<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_tuple<V>(self, _len: usize, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_seq(visitor)
    }

    fn deserialize_tuple_struct<V>(
        self,
        _name: &'static str,
        _len: usize,
        visitor: V,
    ) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_seq(visitor)
    }

    fn deserialize_map<V>(self, _visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_struct<V>(
        self,
        _name: &'static str,
        _fields: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        match self.current() {
            Some(vm::Value::Node { items, .. }) => {
                let l = self.stack.len();
                self.enter_node(items);
                let m = visitor.visit_map(MapDeserializer::new(self))?;
                self.leave_node();
                self.stack[l - 1].pop();
                Ok(m)
            }
            _ => Err(Error::ExpectedNode),
        }
    }

    fn deserialize_enum<V>(
        self,
        _name: &'static str,
        _variants: &'static [&'static str],
        _visitor: V,
    ) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        unimplemented!()
    }

    fn deserialize_identifier<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_any(visitor)
    }

    fn deserialize_ignored_any<V>(self, visitor: V) -> Result<V::Value>
    where
        V: Visitor<'de>,
    {
        self.deserialize_any(visitor)
    }
}

struct MapDeserializer<'a, 'de: 'a> {
    de: &'a mut Deserializer<'de>,
}

impl<'a, 'de> MapDeserializer<'a, 'de> {
    fn new(de: &'a mut Deserializer<'de>) -> Self {
        Self { de }
    }
}

impl<'de, 'a> MapAccess<'de> for MapDeserializer<'a, 'de> {
    type Error = Error;

    fn next_key_seed<K>(&mut self, seed: K) -> Result<Option<K::Value>>
    where
        K: DeserializeSeed<'de>,
    {
        match self.de.current() {
            None => Ok(None),
            Some(vm::Value::Node { .. }) => {
                let v = seed.deserialize(&mut *self.de)?;
                Ok(Some(v))
            }
            Some(_) => Err(Error::ExpectedNode),
        }
    }

    fn next_value_seed<V>(&mut self, seed: V) -> Result<V::Value>
    where
        V: DeserializeSeed<'de>,
    {
        match self.de.current() {
            Some(vm::Value::Node { items, .. }) => {
                let l = self.de.stack.len();
                let v = if items.len() == 1 && !matches!(items[0], vm::Value::Node { .. }) {
                    self.de.enter_node(items);
                    let v = seed.deserialize(&mut *self.de);
                    self.de.leave_node();
                    self.de.stack[l - 1].pop();
                    v
                } else {
                    seed.deserialize(&mut *self.de)
                };
                v
            }
            _ => Err(Error::ExpectedNode),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use langlang_lib::{compiler, parser};

    #[test]
    fn unpack_flat_struct() {
        #[derive(Debug, serde::Deserialize)]
        struct Player {
            name: String,
            score: i64,
            admin: bool,
        }

        let grammar = "
          player <- name COMMA score COMMA admin (EOF / EOL)
          name   <- [a-zA-Z]+
          score  <- [1-9][0-9]*
          admin  <- TRUE / FALSE

          COMMA  <- ','
          TRUE   <- 'true'
          FALSE  <- 'false'
          EOF    <- '\n'
          EOL    <- !.

          name  _ -> text()
          score _ -> i64(text(), 10)
          admin v -> unwrap(v)
          TRUE  _ -> true
          FALSE _ -> false
          COMMA _ -> skip()
        ";

        let input1 = "Larry,235,true";
        let value1 = run(grammar, input1);
        let player1: Player = from_val(&value1).unwrap();

        assert_eq!("Larry".to_string(), player1.name);
        assert_eq!(235, player1.score);
        assert!(player1.admin);

        let input2 = "Moe,298,false";
        let value2 = run(grammar, input2);
        let player2: Player = from_val(&value2).unwrap();

        assert_eq!("Moe".to_string(), player2.name);
        assert_eq!(298, player2.score);
        assert!(!player2.admin);
    }

    #[test]
    fn unpack_recursive_struct() {
        let grammar = "
          #blog     <- _ post* (!.)^eof
          post     <- _ POST OPEN author title comment* CLOSE
          author   <- AUTHOR OPEN name email CLOSE
          comment  <- COMMENT OPEN author content visible CLOSE
          title    <- TITLE String
          name     <- NAME String
          email    <- EMAIL String
          content  <- CONTENT String
          visible  <- VISIBLE Bool

          String   <- QUOTE (!QUOTE .)* QUOTE _
          Bool     <- TRUE / FALSE

          ID       <- [a-zA-Z_][a-zA-Z0-9_]+ _
          AUTHOR   <- 'author'   _
          COMMENT  <- 'comment'  _
          CONTENT  <- 'content'  _
          EMAIL    <- 'email'    _
          POST     <- 'post'     _
          NAME     <- 'name'     _
          TITLE    <- 'title'    _
          VISIBLE  <- 'visible'  _

          TRUE     <- 'true'     _
          FALSE    <- 'false'    _
          OPEN     <- '{' _
          CLOSE    <- '}' _
          QUOTE    <- [']
          _        <- (' ' / '\t' / '\n' / '\r')*

          _       -> skip()
          AUTHOR  -> skip()
          COMMENT -> skip()
          CONTENT -> skip()
          EMAIL   -> skip()
          NAME    -> skip()
          POST    -> skip()
          QUOTE   -> skip()
          TITLE   -> skip()
          VISIBLE -> skip()
          OPEN    -> skip()
          CLOSE   -> skip()

          ID      -> text()
          String  -> unwrapped(text())
          Bool  v -> unwrapped(v)
          TRUE    -> unwrapped(true)
          FALSE   -> unwrapped(false)
        ";

        #[derive(Debug, serde::Deserialize)]
        struct Author {
            name: String,
            email: String,
        }

        #[derive(Debug, serde::Deserialize)]
        struct Comment {
            author: Author,
            content: String,
            visible: bool,
        }

        #[derive(Debug, serde::Deserialize)]
        struct Post {
            author: Author,
            title: String,
            //comments: Vec<Comment>,
        }

        let input = "
        post {
          author {
            name 'lincoln clarete'
            email 'lincoln@clarete.li'
          }
          title 'a wild journey'
        }
        ";
        let value = run(grammar, input);
        let post: Post = from_val(&value).unwrap();

        assert_eq!("lincoln clarete", post.author.name);
        assert_eq!("lincoln@clarete.li", post.author.email);
        assert_eq!("a wild journey", post.title);
    }

    fn run(grammar: &str, input: &str) -> vm::Value {
        let mut p = parser::Parser::new(grammar);
        let ast = p.parse().unwrap();
        let cc = compiler::Config::default();
        let mut c = compiler::Compiler::new(cc);
        let program = c.compile(ast).unwrap();
        let mut m = vm::VM::new(&program);
        let result = m.run_str(input).unwrap();
        result.unwrap()
    }
}
