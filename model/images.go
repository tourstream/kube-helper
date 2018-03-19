package model

// Manifest of an specific docker image extended with tags information from google registry
type Manifest struct {
	LayerID       string   `json:"layerId"`
	Tags          []string `json:"tag"`
	TimeCreatedMs int64    `json:"timeCreatedMs,string"`
}

// TagCollection contains the Manifests of an docker image and a sorted list of manifests
type TagCollection struct {
	Name            string
	Manifests       map[string]Manifest `json:"manifest"`
	SortedManifests []ManifestPair
}

// ManifestPair is used for the sorted list of manifests
type ManifestPair struct {
	Key   string
	Value Manifest
}
