package cable

import "go.k6.io/k6/js/modules"

func init() {
	modules.Register("k6/x/cable", New())
}

type (
	Cable struct {
		vu modules.VU
	}
	RootModule  struct{}
	CableModule struct {
		*Cable
	}
)

var (
	_ modules.Instance = &CableModule{}
	_ modules.Module   = &RootModule{}
)

func New() *RootModule {
	return &RootModule{}
}

func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &CableModule{Cable: &Cable{vu: vu}}
}

func (c *CableModule) Exports() modules.Exports {
	return modules.Exports{Default: c.Cable}
}
