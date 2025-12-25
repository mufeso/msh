package inventory

// DefaultPort is the standard SSH port.
const DefaultPort = 22

// ApplyDefaults fills empty server fields with the resolved user and port.
//
// The caller resolves user and port from all configuration layers
// (including this inventory's "defaults:" block); a value set on the
// server itself always wins.
//
// The loop uses "for i := range" and inv.Servers[i] to modify the slice
// element; "for _, s := range" would only modify a copy.
func (inv *Inventory) ApplyDefaults(user string, port int) {
	for i := range inv.Servers {
		if inv.Servers[i].Port == 0 {
			inv.Servers[i].Port = port
		}
		if inv.Servers[i].User == "" {
			inv.Servers[i].User = user
		}
	}
}
