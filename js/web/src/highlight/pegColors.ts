export const pegColors = {
    ruleName:  "#61afef",  // blue   — rule definition names (LHS of <-)
    ruleRef:   "#abb2bf",  // gray   — rule references in expressions
    literal:   "#98c379",  // green  — quoted string literals
    charClass: "#e5c07b",  // gold   — character classes [a-z]
    label:     "#c678dd",  // purple — ^labels and @import
    comment:   "#5c6370",  // muted  — // line comments
    operator:  "#56b6c2",  // cyan   — <- / ! & ? * + # .
} as const;

export const pegColorsLight = {
    ruleName:  "#0451a5",  // dark blue
    ruleRef:   "#383a42",  // near-black
    literal:   "#50a14f",  // dark green
    charClass: "#b05f00",  // dark amber
    label:     "#7c3aed",  // dark purple
    comment:   "#7a8191",  // medium gray
    operator:  "#0184bc",  // dark cyan
} as const;
