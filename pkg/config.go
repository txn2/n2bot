package pkg

import (
	"regexp"
	"text/template"

	"strings"

	"github.com/Masterminds/sprig"
)

type ParsedTemplates map[string]map[ContentRule]*template.Template

type CompiledRegexes map[*regexp.Regexp]string

type ContentRule struct {
	Key    string `yaml:"key"`
	Equals string `yaml:"equals"`
}

type Configuration struct {
	Replacements []struct {
		Pattern     string `yaml:"pattern"`
		Replacement string `yaml:"replacement"`
	} `yaml:"replacements"`
	Rules []struct {
		Name        string      `yaml:"name"`
		Description string      `yaml:"description"`
		Producer    string      `yaml:"producer"`
		Template    string      `yaml:"template"`
		ContentRule ContentRule `yaml:"contentRule"`
	} `yaml:"rules"`
	ParsedTemplates ParsedTemplates
	CompiledRegexes CompiledRegexes
}

// Warmup parses templates and compiles regex
func (c *Configuration) Warmup() error {
	c.ParsedTemplates = ParsedTemplates{}

	for _, rule := range c.Rules {
		// cleanup template.

		cleanTemplate := strings.Replace(rule.Template, "\n", " ", -1)
		cleanTemplate = strings.Join(strings.Fields(cleanTemplate), " ")
		cleanTemplate = strings.Replace(cleanTemplate, "\\\\", "\\", -1)

		ruleTemplate, err := template.New(rule.Producer).Funcs(sprig.TxtFuncMap()).Parse(cleanTemplate)
		if err != nil {
			return err
		}
		if c.ParsedTemplates[rule.Producer] == nil {
			c.ParsedTemplates[rule.Producer] = make(map[ContentRule]*template.Template, 0)
		}

		c.ParsedTemplates[rule.Producer][rule.ContentRule] = ruleTemplate
	}

	c.CompiledRegexes = CompiledRegexes{}

	for _, r := range c.Replacements {

		rxp, err := regexp.Compile(r.Pattern)
		if err != nil {
			return err
		}
		c.CompiledRegexes[rxp] = r.Replacement
	}

	return nil
}
