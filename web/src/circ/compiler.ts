import type { LangLangValue, SequenceValue } from "@langlang/react";


const prefix = `const std = @import("std");
const transport = @import("lib/transport.zig");
const memory = @import("lib/memory.zig");

const Circuit = @import("lib/circuit.zig").Circuit;
const Component = @import("lib/circuit.zig").Component;

const log = std.log.scoped(.log);

pub fn main() !void {
    defer memory.deinit();

    var circuit = try Circuit.init();
    defer circuit.deinit();
    `;


function codeGen(circuits: Map<string, string>, connections: Set<[[string, string], [string, string]]>) {
    let code = prefix;

    for (const [identifier, value] of circuits.entries()) {
        code += `const ${identifier} = try circuit.createComponent(.{ ${value} });\n    `
    }

    for (const connection of connections) {
        const [[circuitA, portA], [circuitB, portB]] = connection

        code += `try circuit.connect(${circuitA}, ${portA}, ${circuitB}, ${portB});\n    `
    }

    code += '\n}'

    return code;
}

const builtInsMap = {
    std: {
        pin: '.input_pin_gate = .{}',
        led: '.led = .{}',
        wire: '.wire = .{ .inputs = .{}',
        and: '.and_gate = .{}',
        not: '.not_gate = .{}'
    }
}

function getBuiltIn(sequence: string[]): string {
    let context = builtInsMap;
    const breadCrumbs: string[] = []

    for (const route of sequence) {
        breadCrumbs.push(route);

        if (!(route in context)) {
            throw new Error(`${breadCrumbs.join('.')}`)
        }

        // @ts-expect-error 
        context = context[route];
    }

    return context as unknown as string;
}

const circuits: Map<string, string> = new Map();
const connections: Set<[[string, string], [string, string]]> = new Set();

function parseCircuit(value: SequenceValue): string {
    const [id, _, expression] = value.items;

    const idName = traverse(id);
    const valueData = traverse(expression)

    console.log({ id, valueData });

    circuits.set(idName, valueData);

    return ''
}

function parseAttribute(value: SequenceValue): string {
    const ids = value.items.filter((i) => i.type === 'node' && i.name === 'Id').map(traverse);

    return getBuiltIn(ids);
}


function parseCall(value: SequenceValue): string {
    const [id, _, params] = value.items;

    const identifier = traverse(id);

    const parameters = traverse(params).split(',').map(s => s.trim()).filter(s => s).map(s => s.split('.')) as [[string, string], [string, string]];

    if (identifier === 'connect') {
        connections.add(parameters)
    }

    // const idName = traverse(id);
    // const valueData = traverse(expression)
    return ''
}

function parsePort(value: SequenceValue) {
    const [id, _, addr] = value.items;

    const identifier = traverse(id);
    const address = traverse(addr);


    return `${identifier}.${address}`
}


function traverse(value: LangLangValue): any {
    switch (value.type) {
        case 'node': {
            switch (value.name) {
                case 'Attr':
                    return parseAttribute(value.expr as SequenceValue);
                case 'Circuit':
                    return parseCircuit(value.expr as SequenceValue);
                case 'Call':
                    return parseCall(value.expr as SequenceValue);
                case 'Port':
                    return parsePort(value.expr as SequenceValue)
            }

            return traverse(value.expr);
        }
        case 'sequence': {
            return value.items.map(traverse)
        }
        case 'error': {
            return value.expr ? traverse(value.expr) : '';
        }
        case 'string':
            return value.value
    }
}

export function compileZig(tree: LangLangValue) {
    circuits.clear();
    connections.clear();

    console.log('tree', traverse(tree))



    // return JSON.stringify(tree, null, 2)

    return codeGen(circuits, connections)
}