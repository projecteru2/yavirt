package models

import (
	"time"

	"github.com/projecteru2/yavirt/pkg/idgen"
)

func init() {
	idgen.Setup(0, time.Now())
}
