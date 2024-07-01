package calico

import (
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/projecteru2/yavirt/pkg/test/assert"
)

func TestPoolNameStr(t *testing.T) {
	d := &Driver{
		poolNames: map[string]struct{}{"a": {}, "b": {}, "c": {}},
	}
	ss := d.poolNamesStr()
	l := strings.Split(ss, ", ")
	s1 := mapset.NewSet[string](l...)
	diff := s1.Difference(mapset.NewSet("a", "b", "c"))
	assert.Equal(t, diff.Cardinality(), 0)
}
