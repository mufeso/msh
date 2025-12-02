package inventory

// DefaultPort is the SSH port used when neither the server nor the
// inventory defaults set one.
const DefaultPort = 22

// ApplyDefaults fills empty server fields from inv.Defaults and the
// built-in defaults.
//
// It has a pointer receiver because it modifies the inventory. Note the
// loop: "for i := range" and inv.Servers[i] modify the element in the
// slice. "for _, s := range" would give a copy, and changing s would be
// lost.
func (inv *Inventory) ApplyDefaults() {
	port := inv.Defaults.Port
	if port == 0 {
		port = DefaultPort
	}
	for i := range inv.Servers {
		if inv.Servers[i].Port == 0 {
			inv.Servers[i].Port = port
		}
	}
}
