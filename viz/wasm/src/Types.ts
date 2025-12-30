export type StringValue = {
    type: "string";
    value: string;
}

export type SequenceValue = {
    type: "sequence";
    count: number;
    items: LangLangValue[];
}

export type ErrorValue = {
    type: "error";
    expr?: LangLangValue;
}

export type NodeValue = {
    type: "node";
    name: string;
    expr: LangLangValue;
}

export type LangLangValue = StringValue | SequenceValue | ErrorValue | NodeValue;