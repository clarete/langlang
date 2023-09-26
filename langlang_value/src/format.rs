use crate::value::Value;

pub fn value_fmt0(value: &Value) -> String {
    let mut s = String::new();
    s.push_str(format!("{:#?}", value).as_str());
    s
}

pub fn value_fmt1(value: &Value) -> String {
    let mut s = String::new();
    match value {
        Value::Char(ref v) => s.push(v.value),
        Value::String(ref v) => s.push_str(&v.value),
        Value::Node(ref node) => {
            s.push_str(&node.name);
            s.push('[');
            for i in &node.items {
                s.push_str(value_fmt1(i).as_str())
            }
            s.push(']');
        }
        Value::List(ref list) => {
            s.push('[');
            for c in &list.values {
                s.push_str(value_fmt1(c).as_str())
            }
            s.push(']');
        }
        Value::Error(ref err) => {
            s.push_str("Error[");
            s.push_str(&err.label);
            if let Some(m) = &err.message {
                s.push_str(": ");
                s.push_str(m);
            }
            s.push(']');
        }
    }
    s
}

pub fn value_fmt2(value: &Value) -> String {
    fn f(value: &Value, indent: u16) -> String {
        let mut s = String::new();
        match value {
            Value::Char(v) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push('"');
                match v.value {
                    '\n' => s.push_str("\\n"),
                    vv => s.push(vv),
                }
                s.push('"');
            }
            Value::String(v) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(format!(r"{:#?}", v.value).as_str());
            }
            Value::Node(n) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(&n.name);
                s.push(':');
                s.push(' ');
                s.push('[');
                s.push('\n');
                for i in &n.items {
                    s.push_str(f(i, indent + 1).as_str());
                    s.push('\n');
                }
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push(']');
            }

            Value::List(n) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push('{');
                for c in &n.values {
                    s.push_str(f(c, indent + 1).as_str())
                }
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push('}');
            }
            Value::Error(n) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str("Error{");
                s.push_str(&n.label);
                if let Some(m) = &n.message {
                    s.push_str(": ");
                    s.push_str(m);
                }
                s.push('}');
            }
        }
        s
    }
    f(value, 0)
}

pub fn value_html(value: &Value) -> String {
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
                s.push_str(value_html(i).as_str());
            }
            s.push_str("</span>");
        }
        _ => {}
    }
    s
}
