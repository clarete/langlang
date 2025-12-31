export type StringValue = {
    type: "string";
    value: string;
};

export type SequenceValue = {
    type: "sequence";
    count: number;
    items: Value[];
};

export type ErrorValue = {
    type: "error";
    expr?: Value;
    label?: string;
    message?: string;
};

export type NodeValue = {
    type: "node";
    name: string;
    expr: Value;
};

export type Value = StringValue | SequenceValue | ErrorValue | NodeValue;
