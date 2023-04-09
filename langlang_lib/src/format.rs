use crate::vm::Value;

pub fn value_fmt0(value: &Value) -> String {
    let mut s = String::new();
    s.push_str(format!("{:#?}", value).as_str());
    s
}

pub fn value_fmt1(value: &Value) -> String {
    let mut s = String::new();
    match value {
        Value::Char(v) => s.push(*v),
        Value::String(v) => s.push_str(v),
        Value::Node { name, items } => {
            s.push_str(name);
            s.push('[');
            for i in items {
                s.push_str(value_fmt1(i).as_str())
            }
            s.push(']');
        }
        Value::List(items) => {
            s.push('[');
            for c in items {
                s.push_str(value_fmt1(c).as_str())
            }
            s.push(']');
        }
        Value::Error { label, message } => {
            s.push_str("Error[");
            s.push_str(label);
            if let Some(m) = message {
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
                match *v {
                    '\n' => s.push_str("\\n"),
                    vv => s.push(vv),
                }
                s.push('"');
            }
            Value::String(v) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(format!(r"{:#?}", v).as_str());
            }
            Value::Node { name, items } => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(name);
                s.push(':');
                s.push(' ');
                s.push('[');
                s.push('\n');
                for i in items {
                    s.push_str(f(i, indent + 1).as_str());
                    s.push('\n');
                }
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push(']');
            }

            Value::List(items) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push('{');
                for c in items {
                    s.push_str(f(c, indent + 1).as_str())
                }
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push('}');
            }
            Value::Error { label, message } => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str("Error{");
                s.push_str(label);
                if let Some(m) = message {
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
        Value::Char(v) => match *v {
            '\n' => s.push_str("\\n"),
            vv => s.push(vv),
        },
        Value::String(v) => s.push_str(v),
        Value::Node { name, items } => {
            s.push_str("<span class=\"");
            s.push_str(name);
            s.push_str("\">");
            for i in items {
                s.push_str(value_html(i).as_str());
            }
            s.push_str("</span>");
        }
        _ => {}
    }
    s
}
