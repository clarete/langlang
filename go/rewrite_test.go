package langlang

import "testing"

func TestCheckWellFormed(t *testing.T) {
	t.Run("valid rule", func(t *testing.T) {
		rule := &RewriteRule{
			Name:    "swap",
			Pattern: PatNamed{NodeName: "Add", Body: PatVar{Name: "x"}},
			Constr:  ConNamed{NodeName: "Sub", Body: ConVar{Name: "x"}},
		}
		if err := CheckWellFormed(rule); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unbound variable", func(t *testing.T) {
		rule := &RewriteRule{
			Name:    "bad",
			Pattern: PatNamed{NodeName: "A", Body: PatWild{}},
			Constr:  ConVar{Name: "x"},
		}
		err := CheckWellFormed(rule)
		if err == nil {
			t.Fatal("expected error for unbound variable")
		}
	})
}

func TestPatternVars(t *testing.T) {
	p := PatNamed{
		NodeName: "Add",
		Body: PatSeq{Elems: []RewritePattern{
			PatVar{Name: "left"},
			PatVar{Name: "right"},
		}},
	}
	vars := PatternVars(p)
	if len(vars) != 2 || vars[0] != "left" || vars[1] != "right" {
		t.Fatalf("expected [left, right], got %v", vars)
	}
}

func TestConstructionVars(t *testing.T) {
	c := ConNamed{
		NodeName: "Sub",
		Body: ConSeq{Elems: []RewriteConstruction{
			ConVar{Name: "right"},
			ConVar{Name: "left"},
		}},
	}
	vars := ConstructionVars(c)
	if len(vars) != 2 || vars[0] != "right" || vars[1] != "left" {
		t.Fatalf("expected [right, left], got %v", vars)
	}
}

func TestCompileAndRunRewrite(t *testing.T) {
	// Build a tree: Node("Wrapper", String("hello"))
	tr := &tree{
		strs: []string{"Wrapper", "hello", "Renamed"},
	}
	strID := tr.AddString(0, 5)   // "hello" at bytes 0..5
	rootID := tr.AddNode(0, strID, 0, 5) // Node("Wrapper", ...)
	tr.root = rootID
	tr.input = []byte("hello")

	// Rule: Wrapper(?x) -> Renamed(?x)
	rule := &RewriteRule{
		Name: "rename",
		Pattern: PatNamed{
			NodeName: "Wrapper",
			Body:     PatVar{Name: "x"},
		},
		Constr: ConNamed{
			NodeName: "Renamed",
			Body:     ConVar{Name: "x"},
		},
	}

	if err := CheckWellFormed(rule); err != nil {
		t.Fatal(err)
	}

	// Compile the rule
	c := newCompiler(NewConfig())
	c.code = append(c.code, ILabel{ID: 0})

	ruleSet := &RewriteRuleSet{
		Name:  "rename",
		Rules: []*RewriteRule{rule},
	}
	if err := CompileRewriteRuleSet(ruleSet, c); err != nil {
		t.Fatal(err)
	}

	c.emit(IHalt{})

	// Build the program
	p := &Program{
		code:       c.code,
		strings:    c.strings,
		stringsMap: c.stringsMap,
	}

	// Encode to bytecode
	cfg := NewConfig()
	bytecode := Encode(p, cfg)

	// Create VM and rewrite
	vm := NewVirtualMachine(bytecode)
	resultID, err := vm.Rewrite(tr, rootID)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	if tr.Type(resultID) != NodeType_Node {
		t.Fatalf("expected NodeType_Node, got %s", tr.Type(resultID))
	}
	if tr.Name(resultID) != "Renamed" {
		t.Fatalf("expected name 'Renamed', got %q", tr.Name(resultID))
	}
}

func TestStrategyTry(t *testing.T) {
	// try(fail) = choice(fail, id) should succeed with input unchanged
	c := newCompiler(NewConfig())
	c.code = append(c.code, ILabel{ID: 0})
	strat := StratTry{Inner: StratFail{}}
	if err := CompileStrategy(strat, c); err != nil {
		t.Fatal(err)
	}
	c.emit(IHalt{})

	hasFail := false
	hasChoice := false
	hasBuildCopy := false
	for _, inst := range c.code {
		switch inst.(type) {
		case IFail:
			hasFail = true
		case IChoice:
			hasChoice = true
		case IBuildCopy:
			hasBuildCopy = true
		}
	}
	if !hasFail || !hasChoice || !hasBuildCopy {
		t.Fatal("try(fail) should compile to choice/fail/build_copy instructions")
	}
}

func TestStrategyNot(t *testing.T) {
	// not(fail) should succeed: compiles to choice_pred; fail; fail_twice; build_copy
	c := newCompiler(NewConfig())
	c.code = append(c.code, ILabel{ID: 0})
	strat := StratNot{Inner: StratFail{}}
	if err := CompileStrategy(strat, c); err != nil {
		t.Fatal(err)
	}
	c.emit(IHalt{})

	hasChoicePred := false
	hasFailTwice := false
	for _, inst := range c.code {
		switch inst.(type) {
		case IChoicePred:
			hasChoicePred = true
		case IFailTwice:
			hasFailTwice = true
		}
	}
	if !hasChoicePred || !hasFailTwice {
		t.Fatal("not(fail) should compile to choice_pred/fail_twice")
	}
}

func TestStrategyRepeat(t *testing.T) {
	c := newCompiler(NewConfig())
	c.code = append(c.code, ILabel{ID: 0})
	strat := StratRepeat{Inner: StratId{}}
	if err := CompileStrategy(strat, c); err != nil {
		t.Fatal(err)
	}
	c.emit(IHalt{})

	// Should have choice and commit (loop structure)
	choices := 0
	commits := 0
	for _, inst := range c.code {
		switch inst.(type) {
		case IChoice:
			choices++
		case ICommit:
			commits++
		}
	}
	if choices == 0 || commits == 0 {
		t.Fatal("repeat should compile to choice/commit loop")
	}
}

func TestCompileChoiceRewrite(t *testing.T) {
	// Rule set with two alternatives:
	//   fold <~ Add(Num("0"), ?e) -> ?e
	//        /  Add(?e, Num("0")) -> ?e

	rule1 := &RewriteRule{
		Name: "fold",
		Pattern: PatNamed{
			NodeName: "Add",
			Body: PatSeq{Elems: []RewritePattern{
				PatNamed{NodeName: "Num", Body: PatStr{Text: "0"}},
				PatVar{Name: "e"},
			}},
		},
		Constr: ConVar{Name: "e"},
	}

	rule2 := &RewriteRule{
		Name: "fold",
		Pattern: PatNamed{
			NodeName: "Add",
			Body: PatSeq{Elems: []RewritePattern{
				PatVar{Name: "e"},
				PatNamed{NodeName: "Num", Body: PatStr{Text: "0"}},
			}},
		},
		Constr: ConVar{Name: "e"},
	}

	for _, rule := range []*RewriteRule{rule1, rule2} {
		if err := CheckWellFormed(rule); err != nil {
			t.Fatalf("well-formedness check failed: %v", err)
		}
	}

	ruleSet := &RewriteRuleSet{
		Name:  "fold",
		Rules: []*RewriteRule{rule1, rule2},
	}

	c := newCompiler(NewConfig())
	c.code = append(c.code, ILabel{ID: 0})
	if err := CompileRewriteRuleSet(ruleSet, c); err != nil {
		t.Fatalf("compilation failed: %v", err)
	}
	c.emit(IHalt{})

	// Verify the instruction sequence contains choice/commit
	hasChoice := false
	hasCommit := false
	for _, inst := range c.code {
		switch inst.(type) {
		case IChoice:
			hasChoice = true
		case ICommit:
			hasCommit = true
		}
	}

	if !hasChoice || !hasCommit {
		t.Fatal("expected choice/commit instructions for multi-rule set")
	}
}
