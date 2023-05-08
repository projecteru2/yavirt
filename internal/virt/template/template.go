package template

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	text "text/template"

	"github.com/projecteru2/yavirt/pkg/errors"
)

// Render .
func Render(filepath string, defaultTemplStr string, args any) ([]byte, error) {
	var tmpl, err = templates.get(filepath, defaultTemplStr)
	if err != nil {
		return nil, errors.Annotatef(err, "get template %s failed", filepath)
	}

	var wr bytes.Buffer
	if err := tmpl.Execute(&wr, args); err != nil {
		return nil, errors.Trace(err)
	}

	return wr.Bytes(), nil
}

var templates Templates

// Templates .
type Templates struct {
	sync.Map
}

func (t *Templates) get(fpth string, defaultTemplStr string) (tmpl *text.Template, err error) {
	var val, ok = t.Load(fpth)
	if ok {
		return val.(*text.Template), nil
	}

	tmpl, err = t.parse(fpth, defaultTemplStr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	t.Store(fpth, tmpl)

	return tmpl, nil
}

func (t *Templates) parse(fpth string, defaultTemplStr string) (*text.Template, error) {
	if !t.isTempl(fpth) {
		return nil, errors.Errorf("%s is not a template file", fpth)
	}
	f, err := os.Open(fpth)
	// if file path doesn't exist, then use default template string
	if err != nil && os.IsNotExist(err) {
		if defaultTemplStr == "" {
			return nil, fmt.Errorf("Can't render %s: file doesn't exist and defaut template string is empty", fpth)
		}
		return text.New(fpth).Parse(defaultTemplStr)
	}
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(f)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return text.New(fpth).Parse(string(buf))
}

func (t *Templates) isTempl(fpth string) bool {
	return strings.HasSuffix(fpth, ".xml")
}
