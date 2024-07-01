package ovn

type LogicalSwitch struct {
	UUID   string            `ovsdb:"_uuid"` // _uuid tag is mandatory
	Name   string            `ovsdb:"name"`
	Ports  []string          `ovsdb:"ports"`
	Config map[string]string `ovsdb:"other_config"`
}

type LogicalSwitchPort struct {
	UUID             string            `ovsdb:"_uuid"`
	Name             string            `ovsdb:"name"`
	Type             string            `ovsdb:"type"`
	ExternalIDs      map[string]string `ovsdb:"external_ids"`
	Options          map[string]string `ovsdb:"options"`
	Addresses        []string          `ovsdb:"addresses"`
	PortSecurity     []string          `ovsdb:"port_security"`
	DynamicAddresses *string           `ovsdb:"dynamic_addresses"`
	ParentName       *string           `ovsdb:"parent_name"`
	Tag              *int              `ovsdb:"tag"`
	TagRequest       *int              `ovsdb:"tag_request"`
	Up               *bool             `ovsdb:"up"`
}

type Interface struct {
	UUID        string            `ovsdb:"_uuid"`
	Name        string            `ovsdb:"name"`
	Type        string            `ovsdb:"type"`
	Error       *string           `ovsdb:"error"`
	ExternalIDs map[string]string `ovsdb:"external_ids"`
	Statistics  map[string]int    `ovsdb:"statistics"`
	Config      map[string]string `ovsdb:"other_config"`
	AdminState  *string           `ovsdb:"admin_state"`
	LinkState   *string           `ovsdb:"link_state"`
}
