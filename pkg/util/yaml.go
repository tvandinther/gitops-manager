package util

import (
	"sort"

	"gopkg.in/yaml.v3"
)

func DeleteMappingKeysByIndices(mapping *yaml.Node, indices []int) {
	// Sort in descending order to avoid shifting
	sort.Slice(indices, func(i, j int) bool { return indices[i] > indices[j] })

	for _, idx := range indices {
		if idx%2 != 0 || idx < 0 || idx+1 >= len(mapping.Content) {
			continue // skip invalid map key positions
		}
		mapping.Content = append(mapping.Content[:idx], mapping.Content[idx+2:]...)
	}
}

func SetMappingValue(mapping *yaml.Node, key string, value string) {
	for i := 0; i < len(mapping.Content); i += 2 {
		k := mapping.Content[i]
		v := mapping.Content[i+1]
		if k.Value == key {
			v.Kind = yaml.ScalarNode
			v.Value = value
			return
		}
	}
	// Key not found, append
	mapping.Content = append(mapping.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: key,
		Tag:   "!!str",
	}, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
		Tag:   "!!str",
	})
}

func GetOrCreateMap(parent *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(parent.Content); i += 2 {
		k := parent.Content[i]
		v := parent.Content[i+1]
		if k.Value == key {
			if v.Kind != yaml.MappingNode {
				v.Kind = yaml.MappingNode
				v.Content = []*yaml.Node{}
			}
			return v
		}
	}
	// Key not found, create new
	newMap := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{},
	}
	parent.Content = append(parent.Content, &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: key,
		Tag:   "!!str",
	}, newMap)

	return newMap
}
