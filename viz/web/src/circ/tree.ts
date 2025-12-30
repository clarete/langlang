import type { LangLangValueType, LangLangValue, NodeValue, SequenceValue, StringValue, GenericValue, ErrorValue } from "@langlang/react";
import type { Simplify } from "type-fest";

export type ReferenceNode = LangLangValueType<'ref', {
    address: string[];
    port?: string;
}>;

export type CircuitNode = LangLangValueType<'circuit', {
    identifier: string,
    value: ReferenceNode,
}>;

export type ProgramNode = LangLangValueType<'program', {
    texts: LangLangValue[],
}>;

export type ParamsNode = LangLangValueType<'params', {
    items: ReferenceNode[],
}>;

export type CallNode = LangLangValueType<'call', {
    name: string;
    parameters: ReferenceNode[];
}>;

export type AbstractSyntaxTree = Simplify<LangLangValue | ProgramNode | CircuitNode | ParamsNode | CallNode | ReferenceNode>;

type NodeVisitor<Value extends GenericValue> = (node: NodeValue, traverse: (value: Value) => Value) => Value;


function assertReferenceValue(value: GenericValue): asserts value is ReferenceNode {
    if (value.type !== 'ref') {
        throw new Error('Expected reference node');
    }
}

export function traverseTree<Value extends GenericValue>(value: Value, onNode: NodeVisitor<Value>): Value {
    switch (value.type) {
        case 'node': {
            assertNodeValue(value);
            return onNode(value, (v: Value) => traverseTree(v, onNode));
        }
        case 'sequence': {
            assertSequenceValue(value);
            return { ...value, items: value.items.map(item => traverseTree(item as Value, onNode)) };
        }
        case 'error': {
            assertErrorValue(value);
            if (value.expr) {
                return traverseTree(value.expr as Value, onNode);
            }
            throw new Error('Error node without expression');
        }
        case 'string':
            assertStringValue(value);
            return { type: 'string', value: value.value } as unknown as Value;
        default:
            return value;
    }
}

const translateProgram: NodeVisitor<AbstractSyntaxTree> = (node, traverse): AbstractSyntaxTree => {

    const { expr } = node;
    assertSequenceValue(expr);

    const texts = expr.items.map(traverse) as LangLangValue[];

    return {
        type: 'program',
        texts
    };
}

const translateAttribute: NodeVisitor<AbstractSyntaxTree> = (node): AbstractSyntaxTree => {
    const { expr } = node;
    assertSequenceValue(expr);

    const address = expr.items.filter((i) => i.type === 'node' && i.name === 'Id').map((n) => readIdentifier(n));

    return {
        type: 'ref',
        address,
    }
}

function assertCircuitItems(items: AbstractSyntaxTree[]): asserts items is [NodeValue, StringValue, ReferenceNode] {
    if (items.length < 3) {
        throw new Error('Circuit node must have at least 3 items');
    }

    const [IdNode, stringNode, referenceNode] = items;

    assertNodeValue(IdNode);
    assertStringValue(stringNode);
    assertReferenceValue(referenceNode);
}

const translateCircuit: NodeVisitor<AbstractSyntaxTree> = (node, traverse): AbstractSyntaxTree => {
    const { expr } = node;
    assertSequenceValue(expr);

    const { items } = traverse(expr) as SequenceValue;

    assertCircuitItems(items);

    const [idNode, _, referenceNode] = items;

    return {
        type: 'circuit',
        identifier: readIdentifier(idNode),
        value: referenceNode,
    };
}

function assertPortItems(items: AbstractSyntaxTree[]): asserts items is [NodeValue, StringValue, NodeValue, StringValue] {
    if (items.length < 4) {
        throw new Error('Circuit node must have at least 4 items');
    }

    const [IdNode, sqrBrackOpn, portNum, sqrBrackCls] = items;

    assertNodeValue(IdNode);
    assertStringValue(sqrBrackOpn);
    assertNodeValue(portNum);
    assertStringValue(sqrBrackCls);
}

const translatePort: NodeVisitor<AbstractSyntaxTree> = (node, traverse): AbstractSyntaxTree => {
    const { expr } = node;
    assertSequenceValue(expr);

    assertPortItems(expr.items);

    const [idNode, , portNumNode] = expr.items;

    const portStringNode = traverse(portNumNode)

    assertStringValue(portStringNode);

    return {
        type: 'ref',
        address: [readIdentifier(idNode)],
        port: readStringValue(portStringNode),
    }
}

function assertCallItems(items: AbstractSyntaxTree[]): asserts items is [NodeValue, StringValue, NodeValue, StringValue] {
    if (items.length < 4) {
        throw new Error('Circuit node must have at least 4 items');
    }

    const [IdNode, prntOpn, paramsNode, prntCls] = items;

    assertNodeValue(IdNode);
    assertStringValue(prntOpn);
    assertNodeValue(paramsNode);
    assertStringValue(prntCls);
}

const translateCall: NodeVisitor<AbstractSyntaxTree> = (node, traverse): AbstractSyntaxTree => {
    const { expr } = node;
    assertSequenceValue(expr);


    assertCallItems(expr.items);

    const [idNode, , paramsNode] = expr.items;

    const name = readIdentifier(idNode);
    const parameters = traverse(paramsNode) as ParamsNode;

    return {
        type: 'call',
        name,
        parameters: parameters.items,
    };
}

const translateParams: NodeVisitor<AbstractSyntaxTree> = (node, traverse): AbstractSyntaxTree => {
    const { expr } = node;
    assertSequenceValue(expr);

    const items = expr.items.filter(({ type }) => type === 'node').map(traverse) as ReferenceNode[];

    return {
        type: 'params',
        items,
    };
}

export function translate(tree: LangLangValue) {
    return traverseTree(tree as AbstractSyntaxTree, (node, traverse) => {
        switch (node.name) {
            case 'Program':
                return translateProgram(node, traverse);
            case 'Attr':
                return translateAttribute(node, traverse);
            case 'Circuit':
                return translateCircuit(node, traverse);
            case 'Port':
                return translatePort(node, traverse);
            case 'Call':
                return translateCall(node, traverse);
            case 'Params':
                return translateParams(node, traverse);
            case 'Expr':
            case 'Multi':
            case 'Primary':
            case 'Text':
            case 'Num':
                return traverse(node.expr);
            default:
                return {
                    ...node,
                    expr: traverse(node.expr)
                } as AbstractSyntaxTree;
        }
    })
}

export function readStringValue(value: LangLangValue) {
    if (value.type !== 'string') {
        throw new Error('Expected string value');
    }

    return value.value;
}

export function readIdentifier(value: LangLangValue) {
    if (value.type !== 'node' || value.name !== 'Id') {
        throw new Error('Expected identifier node');
    }

    return readStringValue(value.expr);
}

export function assertSequenceValue(value: GenericValue): asserts value is SequenceValue {
    if (value.type !== 'sequence') {
        throw new Error('Expected sequence value');
    }
}

export function assertNodeValue(value: GenericValue): asserts value is NodeValue {
    if (value.type !== 'node') {
        throw new Error('Expected node value');
    }
}

export function assertErrorValue(value: GenericValue): asserts value is ErrorValue {
    if (value.type !== 'error') {
        throw new Error('Expected error value');
    }
}

export function assertStringValue(value: GenericValue): asserts value is StringValue {
    if (value.type !== 'string') {
        throw new Error('Expected string value');
    }
}

export function assertCircuitNode(value: AbstractSyntaxTree): asserts value is CircuitNode {
    if (value.type !== 'circuit') {
        throw new Error('Expected circuit node');
    }
}

export function assertProgramNode(value: AbstractSyntaxTree): asserts value is ProgramNode {
    if (value.type !== 'program') {
        throw new Error('Expected program node');
    }
}

export function assertCallNode(value: AbstractSyntaxTree): asserts value is CallNode {
    if (value.type !== 'call') {
        throw new Error('Expected call node');
    }
}

export function assertParamsNode(value: AbstractSyntaxTree): asserts value is ParamsNode {
    if (value.type !== 'params') {
        throw new Error('Expected params node');
    }
}

export function assertReferenceNode(value: AbstractSyntaxTree): asserts value is ReferenceNode {
    if (value.type !== 'ref') {
        throw new Error('Expected reference node');
    }
}

