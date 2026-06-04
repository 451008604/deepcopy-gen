package iface

type WithInterface struct {
	Name   string
	Data   interface{}
	Config interface{}
}

type InterfaceSlice struct {
	Items []interface{}
}

type InterfaceMap struct {
	Values map[string]interface{}
}

type NestedInterface struct {
	Meta map[string]interface{}
	Tags []interface{}
}
