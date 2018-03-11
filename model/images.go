package model

type Manifest struct {
	LayerID       string   `json:"layerId"`
	Tags          []string `json:"tag"`
	TimeCreatedMs int64    `json:"timeCreatedMs,string"`
}

type TagCollection struct {
	Name            string
	Manifests       map[string]Manifest `json:"manifest"`
	SortedManifests []ManifestPair
}

type ManifestPair struct {
	Key   string
	Value Manifest
}
