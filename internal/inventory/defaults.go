package inventory

// DefaultPort is the SSH port used when neither the server nor the
// inventory defaults set one.
const DefaultPort = 22

// ApplyDefaults fills empty server fields.
//
// Precedence for each field: the server's own value, then inv.Defaults,
// then the built-in default. For the user, the built-in default is
// fallbackUser (normally the local login name), passed in by the caller so
// that this package never reads the environment itself.
//
// The loop uses "for i := range" and inv.Servers[i] to modify the slice
// element; "for _, s := range" would only modify a copy.
func (inv *Inventory) ApplyDefaults(fallbackUser string) {
	port := inv.Defaults.Port
	if port == 0 {
		port = DefaultPort
	}
	user := inv.Defaults.User
	if user == "" {
		user = fallbackUser
	}
	for i := range inv.Servers {
		if inv.Servers[i].Port == 0 {
			inv.Servers[i].Port = port
		}
		if inv.Servers[i].User == "" {
			inv.Servers[i].User = user
		}
	}
}
