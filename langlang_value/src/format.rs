use crate::value::{self, Value};
use crate::visitor::{walk_list, walk_node, Visitor};

// The raw formater uses the host language's formatting function
pub fn raw(value: &Value) -> String {
    format!("{:#?}", value)
}

// compact formatter wraps lists and nodes around square brackets
pub fn compact(value: &Value) -> String {
    let mut f = CompactFormatter::default();
    f.visit_value(value);
    f.output
}

// The indented formatter will print out values spanning multiple
// lines if container objects like lists or nodes are present
pub fn indented(value: &Value) -> String {
    let mut f = IndentedFormatter::default();
    f.visit_value(value);
    f.output
}

// The html formatter will wrapp all node objects around a span tag
// with containing a class attribute that's named after the node.
pub fn html(value: &Value) -> String {
    let mut s = String::new();
    match value {
        Value::Char(v) => match v.value {
            '\n' => s.push_str("\\n"),
            vv => s.push(vv),
        },
        Value::String(v) => s.push_str(&v.value),
        Value::Node(node) => {
            s.push_str("<span class=\"");
            s.push_str(&node.name);
            s.push_str("\">");
            for i in &node.items {
                s.push_str(html(i).as_str());
            }
            s.push_str("</span>");
        }
        _ => {}
    }
    s
}

#[derive(Default)]
struct CompactFormatter {
    output: String,
}

impl<'a> Visitor<'a> for CompactFormatter {
    fn visit_char(&mut self, n: &'a value::Char) {
        self.output.push(n.value);
    }

    fn visit_string(&mut self, n: &'a value::String) {
        self.output.push_str(&n.value);
    }

    fn visit_list(&mut self, n: &'a value::List) {
        self.output.push('[');
        walk_list(self, n);
        self.output.push(']');
    }

    fn visit_node(&mut self, n: &'a value::Node) {
        self.output.push_str(&n.name);
        self.output.push('[');
        walk_node(self, n);
        self.output.push(']');
    }

    fn visit_error(&mut self, n: &'a value::Error) {
        self.output.push_str("Error[");
        self.output.push_str(&n.label);
        if let Some(m) = &n.message {
            self.output.push_str(": ");
            self.output.push_str(m);
        }
        self.output.push(']');
    }
}

#[derive(Default)]
struct IndentedFormatter {
    output: String,
    depth: usize,
}

impl IndentedFormatter {
    fn indent(&mut self) {
        self.depth += 1
    }

    fn unindent(&mut self) {
        self.depth -= 1
    }

    fn write_indent(&mut self) {
        for _ in 0..self.depth {
            self.output.push_str("    ")
        }
    }

    fn writes(&mut self, v: &str) {
        self.write_indent();
        self.output.push_str(v)
    }
}

impl<'a> Visitor<'a> for IndentedFormatter {
    fn visit_char(&mut self, n: &'a value::Char) {
        self.writes(&format!("'{}'\n", n.value));
    }

    fn visit_string(&mut self, n: &'a value::String) {
        self.writes(&format!("'{}'\n", n.value));
    }

    fn visit_list(&mut self, n: &'a value::List) {
        self.writes("{\n");
        self.indent();
        walk_list(self, n);
        self.unindent();
        self.writes("}\n");
    }

    fn visit_node(&mut self, n: &'a value::Node) {
        self.writes(&n.name);
        self.output.push_str(" {\n");
        self.indent();
        walk_node(self, n);
        self.unindent();
        self.writes("}\n");
    }

    fn visit_error(&mut self, n: &'a value::Error) {
        self.writes("Error{");
        self.output.push_str(&n.label);
        if let Some(m) = &n.message {
            self.output.push_str(": ");
            self.output.push_str(m);
        }
        self.output.push_str("}");
    }
}
