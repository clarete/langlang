package langlang

import (
	"fmt"
	"sort"
)

type Config map[string]*cfgVal

// NewConfig creates a new configuration object primed with all the
// default values expected by both the grammar loader and the
// compiler.
func NewConfig() *Config {
	m := make(Config)
	m.SetBool("grammar.add_builtins", true)
	m.SetBool("grammar.add_charsets", true)
	m.SetBool("grammar.captures", true)
	m.SetBool("grammar.capture_spaces", true)
	m.SetBool("grammar.handle_spaces", true)
	m.SetInt("compiler.optimize", 1)
	return &m
}

func (c *Config) Debug() {
	fmt.Println("Configuration")

	keys := make([]string, 0, len(*c))
	width := 0
	for k := range *c {
		keys = append(keys, k)
		width = max(width, len(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf(k)
		for i := 0; i < width-len(k); i++ {
			fmt.Printf(" ")
		}
		fmt.Printf(" : ")
		fmt.Println((*c)[k].String())
	}
}

type cfgValType int

const (
	cfgValType_Undefined cfgValType = iota
	cfgValType_Bool
	cfgValType_Int
	cfgValType_String
)

func (vt cfgValType) String() string {
	return map[cfgValType]string{
		cfgValType_Undefined: "undefined",
		cfgValType_Bool:      "bool",
		cfgValType_Int:       "int",
		cfgValType_String:    "string",
	}[vt]
}

type cfgVal struct {
	typ      cfgValType
	asBool   bool
	asInt    int
	asString string
}

// assignType is mostly for preventing programming errors, it
func (v *cfgVal) assignType(vt cfgValType) {
	if v.typ != vt && v.typ != cfgValType_Undefined {
		panic(fmt.Sprintf("Can't assign `%s` to type `%s`", vt, v.typ))
	}
	v.typ = vt
}

func (v *cfgVal) checkType(vt cfgValType) {
	if v.typ != vt {
		panic(fmt.Sprintf("Can't retrieve `%s` from `%s` variable", vt, v.typ))
	}
}

func (v *cfgVal) String() string {
	switch v.typ {
	case cfgValType_Bool:
		return fmt.Sprintf("%t (bool)", v.asBool)
	case cfgValType_Int:
		return fmt.Sprintf("%d (int)", v.asInt)
	case cfgValType_String:
		return fmt.Sprintf("%s (string)", v.asString)
	case cfgValType_Undefined:
		return "(undefined)"
	default:
		panic(fmt.Sprintf("unknown cfgVal type: %v", v.typ))
	}
}

func (c *Config) SetBool(path string, v bool) {
	(*c)[path] = &cfgVal{}
	(*c)[path].assignType(cfgValType_Bool)
	(*c)[path].asBool = v
}

func (c *Config) SetInt(path string, v int) {
	(*c)[path] = &cfgVal{}
	(*c)[path].assignType(cfgValType_Int)
	(*c)[path].asInt = v
}

func (c *Config) SetString(path string, v string) {
	(*c)[path] = &cfgVal{}
	(*c)[path].assignType(cfgValType_String)
	(*c)[path].asString = v
}

func (c *Config) GetBool(path string) bool {
	if val, ok := (*c)[path]; ok {
		val.checkType(cfgValType_Bool)
		return val.asBool
	}
	panic(fmt.Sprintf("Bool setting `%s` does not exist", path))
}

func (c *Config) GetInt(path string) int {
	if val, ok := (*c)[path]; ok {
		val.checkType(cfgValType_Int)
		return val.asInt
	}
	panic(fmt.Sprintf("Int setting `%s` does not exist", path))
}

func (c *Config) GetString(path string) string {
	if val, ok := (*c)[path]; ok {
		val.checkType(cfgValType_String)
		return val.asString
	}
	panic(fmt.Sprintf("String setting `%s` does not exist", path))
}
