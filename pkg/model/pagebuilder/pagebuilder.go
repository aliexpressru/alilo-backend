package pagebuilder

type Resp struct {
	List []*ListType `json:"list"`
}

type ListType struct {
	Type         string              `json:"type"`
	URL          string              `json:"url"`
	Dependencies []*DependenciesType `json:"dependencies"`
}

type DependenciesType struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Service string `json:"service"`
	Alias   string `json:"alias,omitempty"`
}

func (objs Resp) Len() int {
	return len(objs.List)
}
func (objs Resp) Swap(i, j int) {
	objs.List[i], objs.List[j] = objs.List[j], objs.List[i]
}
func (objs Resp) Less(i, j int) bool {
	return objs.List[i].URL < objs.List[j].URL
}
