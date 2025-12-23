import type { StringValue } from "@langlang/react";

type ReferenceNode = {
    type: 'ref';
    address: string[];
    port?: string;
}

type CircuitNode = {
    type: 'circuit',
    identification: ReferenceNode,
    value: ReferenceNode,
}

export type CircAst = ReferenceNode | CircuitNode;


export function readStringValue(value: StringValue) {

}