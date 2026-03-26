package tomlextract

//go:generate langlang extract -grammar=toml.peg

type TOMLDoc struct {
	Expressions []TOMLExpression `ll:"Expression"`
}

type TOMLExpression struct {
	Table  *TOMLTable  `ll:"Table"`
	KeyVal *TOMLKeyVal `ll:"KeyVal"`
}

type TOMLTable struct {
	Key     TOMLKey       `ll:"Key"`
	KeyVals []TOMLKeyVal  `ll:"KeyVal"`
}

type TOMLKeyVal struct {
	Key TOMLKey `ll:"Key"`
	Val TOMLVal `ll:"Val"`
}

type TOMLKey struct {
	SimpleKeys []TOMLSimpleKey `ll:"SimpleKey"`
}

type TOMLSimpleKey struct {
	BareKey    *string `ll:"BareKey"`
	QuotedKey  *string `ll:"BasicString"`
}

type TOMLVal struct {
	InlineTable *TOMLInlineTable `ll:"InlineTable"`
	Array       *TOMLArray       `ll:"Array"`
	String      *string          `ll:"BasicString"`
	Number      *string          `ll:"Number"`
	Boolean     *string          `ll:"Boolean"`
	DateTime    *string          `ll:"DateTime"`
}

type TOMLArray struct {
	Items []TOMLVal `ll:"Val"`
}

type TOMLInlineTable struct {
	KeyVals []TOMLInlineKeyVal `ll:"InlineKeyVal"`
}

type TOMLInlineKeyVal struct {
	Key TOMLKey `ll:"Key"`
	Val TOMLVal `ll:"Val"`
}
