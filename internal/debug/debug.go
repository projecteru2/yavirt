package debug

import (
	"encoding/json"
	"net/http"

	"github.com/projecteru2/yavirt/internal/eru/resources"
	"github.com/projecteru2/yavirt/internal/vmcache"
)

func Handler(w http.ResponseWriter, _ *http.Request) {
	infos := vmcache.FetchDomainsInfo()
	resp := map[string]any{
		"infos": infos,
		"gpu": map[string]any{
			"capacity": resources.GetManager().FetchGPU(),
		},
		"cpumem": resources.GetManager().FetchCPUMem(),
	}
	bs, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")

	_, _ = w.Write(bs)
}
