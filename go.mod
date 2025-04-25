module src.agwa.name/snid

go 1.23.0

toolchain go1.24.1

require (
	github.com/Tnze/go-mc v1.20.2
	github.com/vishvananda/netlink v1.3.0
	golang.org/x/sys v0.32.0
	golang.org/x/text v0.12.0
	src.agwa.name/go-listener v0.6.1
)

require (
	github.com/google/uuid v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
)

replace src.agwa.name/go-listener => github.com/Arusekk/go-listener v0.0.0-20250425101201-dc236f751a85
