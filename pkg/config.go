package pkg

import (
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig"
)

type ParsedTemplates map[string]map[ContentRule]*template.Template

type CompiledRegexes map[string]*regexp.Regexp

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
		ruleTemplate, err := template.New(rule.Producer).Funcs(sprig.TxtFuncMap()).Parse(rule.Template)
		if err != nil {
			return err
		}
		if c.ParsedTemplates[rule.Producer] == nil {
			c.ParsedTemplates[rule.Producer] = make(map[ContentRule]*template.Template, 0)
		}

		c.ParsedTemplates[rule.Producer][rule.ContentRule] = ruleTemplate
	}

	c.CompiledRegexes = make(map[string]*regexp.Regexp)

	for _, r := range c.Replacements {
		rxp, err := regexp.Compile(r.Pattern)
		if err != nil {
			return err
		}
		c.CompiledRegexes[r.Replacement] = rxp
	}

	return nil
}
