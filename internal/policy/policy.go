package policy

type Action string

const (
	Block          Action = "block"
	AllowButWarn   Action = "allow_but_warn"
	AllowButRedact Action = "allow_but_redact"
	Allow          Action = "allow"
)

type CustomRule struct {
	Definition string `json:"definition"`
	Action     Action `json:"action"`
}

type Policy struct {
	Id           string        `json:"id"`
	NameRule     Action        `json:"nameRule"`
	AddressRule  Action        `json:"addressRule"`
	EmailRule    Action        `json:"emailRule"`
	SsnRule      Action        `json:"ssnRule"`
	PasswordRule Action        `json:"passwordRule"`
	CustomRules  []*CustomRule `json:"customRules"`
}
