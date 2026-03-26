package jsonextract

//go:generate langlang extract -grammar=../../../docs/live/assets/examples/json/json.peg

type JSONValue struct {
	Object *JSONObject `ll:"Object"`
	Array  *JSONArray  `ll:"Array"`
	String *string     `ll:"String"`
	Number *string     `ll:"Number"`
}

type JSONObject struct {
	Members []JSONMember `ll:"Member"`
}

type JSONMember struct {
	Key   string    `ll:"String"`
	Value JSONValue `ll:"Value"`
}

type JSONArray struct {
	Items []JSONValue `ll:"Value"`
}
