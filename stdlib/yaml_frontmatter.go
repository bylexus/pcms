package stdlib

import (
	"io/fs"
	"regexp"

	"gopkg.in/yaml.v3"
)

type YamlFrontMatter map[string]any

// Tries to extract a YAML frontmatter from a string.
// A YAML frontmatter is a section at the beginning of the string,
// containing YAML. It is separated by using '---'.
//
// Example:
// ---
// title: foo
// ---
// here comes the rest of the doc...
//
// returns:
// (yaml object, "here comes the rest of the doc...")
func ExtractYamlFrontMatter(doc string) (YamlFrontMatter, string, error) {
	yamlObj := make(YamlFrontMatter)
	// Matches the whole preamble block and the rest of the doc as
	// separate groups:
	preamblePattern := regexp.MustCompile(`(?s)^\s*(-{3,}\n(.*?)\n-{3,}\n)*(.*)$`)
	matches := preamblePattern.FindStringSubmatch(doc)
	if matches != nil {
		yamlDoc := matches[2]
		restDoc := matches[3]
		err := yaml.Unmarshal([]byte(yamlDoc), &yamlObj)
		if err != nil {
			return nil, "", err
		}
		return yamlObj, restDoc, nil
	} else {
		return yamlObj, doc, nil
	}
}

func ExtractYamlFrontMatterFromFS(fsys fs.FS, name string) (YamlFrontMatter, string, error) {
	buffer, err := fs.ReadFile(fsys, name)

	if err != nil {
		return nil, "", err
	}
	doc := string(buffer[:])
	return ExtractYamlFrontMatter(doc)
}
