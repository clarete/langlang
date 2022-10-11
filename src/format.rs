use crate::vm::Value;

pub fn value_fmt0(value: &Value) -> String {
    let mut s = String::new();
    s.push_str(format!("{:#?}", value).as_str());
    s
}

pub fn value_fmt1(value: &Value) -> String {
    let mut s = String::new();
    match value {
        Value::Chr(v) => s.push(*v),
        Value::Str(v) => s.push_str(v),
        Value::Node { name, children } => {
            s.push_str(name);
            s.push('[');
            for c in children {
                s.push_str(value_fmt1(c).as_str())
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
            Value::Chr(v) => {
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
            Value::Str(v) => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(format!(r"{:#?}", v).as_str());
            }
            Value::Node { name, children } => {
                for _ in 0..indent {
                    s.push_str("    ");
                }
                s.push_str(name);
                s.push_str(" {");
                if !children.is_empty() {
                    s.push('\n');
                    for (i, c) in children.iter().enumerate() {
                        s.push_str(f(c, indent + 1).as_str());
                        if i < children.len() {
                            s.push('\n');
                        }
                    }
                    for _ in 0..indent {
                        s.push_str("    ");
                    }
                }
                s.push('}');
            }
        }
        s
    }
    f(value, 0)
}
