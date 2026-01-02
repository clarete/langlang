export type Span = {
    start: Location;
    end: Location;
};

export type Location = {
    line: number;
    column: number;
    cursor: number;
    utf16Cursor: number;
};

export type StringValue = {
    type: "string";
    value: string;
    span: Span;
};

export type SequenceValue = {
    type: "sequence";
    count: number;
    items: Value[];
    span: Span;
};

export type ErrorValue = {
    type: "error";
    expr?: Value;
    label?: string;
    message?: string;
    span: Span;
};

export type NodeValue = {
    type: "node";
    name: string;
    expr: Value;
    span: Span;
};

export type Value = StringValue | SequenceValue | ErrorValue | NodeValue;
