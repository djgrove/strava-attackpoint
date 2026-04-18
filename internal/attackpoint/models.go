package attackpoint

// FormSchema represents the discovered structure of AP's new training form.
type FormSchema struct {
	Action         string                 // form action URL
	Fields         map[string]FormField   // field name → field info
	ActivityTypes  []SelectOption         // options from the activity type select
}

// FormField represents a discovered form field.
type FormField struct {
	Name    string
	Type    string         // "input", "select", "textarea"
	Options []SelectOption // only for select fields
}

// SelectOption is a value/label pair from a <select> element.
type SelectOption struct {
	Value string
	Label string
}
