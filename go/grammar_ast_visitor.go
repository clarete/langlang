package langlang

type AstNodeVisitor interface {
	VisitGrammarNode(*GrammarNode) error
	VisitImportNode(*ImportNode) error
	VisitDefinitionNode(*DefinitionNode) error
	VisitCaptureNode(*CaptureNode) error
	VisitSequenceNode(*SequenceNode) error
	VisitOneOrMoreNode(*OneOrMoreNode) error
	VisitZeroOrMoreNode(*ZeroOrMoreNode) error
	VisitOptionalNode(*OptionalNode) error
	VisitChoiceNode(*ChoiceNode) error
	VisitAndNode(*AndNode) error
	VisitNotNode(*NotNode) error
	VisitLexNode(*LexNode) error
	VisitLabeledNode(*LabeledNode) error
	VisitLiteralNode(*LiteralNode) error
	VisitClassNode(*ClassNode) error
	VisitRangeNode(*RangeNode) error
	VisitAnyNode(*AnyNode) error
	VisitIdentifierNode(*IdentifierNode) error
}

func WalkGrammarNode(g AstNodeVisitor, n *GrammarNode) error {
	for _, item := range n.GetItems() {
		if err := item.Accept(g); err != nil {
			return err
		}
	}
	return nil
}
