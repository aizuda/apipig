package toolkit

import "github.com/flosch/pongo2/v6"

func TextRender(text string, params map[string]any) string {
	tpl, err := pongo2.FromString(text)
	if err == nil {
		out, err := tpl.Execute(params)
		if err == nil {
			return out
		}
	}
	return text
}
