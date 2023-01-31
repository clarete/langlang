mod utils;

use langlang_lib::{compiler, format, parser, vm, Error};
use wasm_bindgen::prelude::*;

// When the `wee_alloc` feature is enabled, use `wee_alloc` as the global
// allocator.
#[cfg(feature = "wee_alloc")]
#[global_allocator]
static ALLOC: wee_alloc::WeeAlloc = wee_alloc::WeeAlloc::INIT;

// #[wasm_bindgen]
// #[repr(u8)]
// #[derive(Clone, Copy, Debug, PartialEq, Eq)]
// pub enum Cell {

// }

// pub struct Viewer {
//     width: u32,
//     height: u32,
//     cells: Vec<Cell>,
// }

// #[wasm_bindgen]
// impl Viewer {
//     fn new(width: u32, height: u32) -> Self {
//         Self {
//             width,
//             height,
//             cells: vec![],
//         }
//     }

//     fn get_index(&self, row: u32, column: u32) -> usize {
//         (row * self.width + column) as usize
//     }
// }

#[wasm_bindgen]
pub struct Lang {
    grammar_prg: vm::Program,
}

#[wasm_bindgen]
impl Lang {
    pub fn new() -> Self {
        let grammar_txt = include_str!("../grammar.peg").to_string();
        let grammar_ast = parser::Parser::new(&grammar_txt).parse().unwrap();
        let grammar_prg = compiler::Compiler::default().compile(grammar_ast).unwrap();
        Self { grammar_prg }
    }

    fn run(&self, input: &str) -> Result<Option<vm::Value>, Error> {
        Ok(vm::VM::new(&self.grammar_prg).run_str(input)?)
    }

    fn pprint(&self, input: &str) -> Result<String, Error> {
        let out = self.run(input)?;
        Ok(format::value_html(&out.unwrap()))
    }

    pub fn highlight(&self, code: &str) -> String {
        match self.pprint(code) {
            Ok(v) => v,
            Err(_) => code.to_string(),
        }
    }
}

impl Default for Lang {
    fn default() -> Self {
        Self::new()
    }
}
