package model

const (
	ArtifactManifestSchemaVersion = 1
	ArtifactTypeContainerImage    = "container_image"
	ArtifactTypeStaticArchive     = "static_archive"
	ArtifactTypeReleaseDirectory  = "release_directory"
)

type ArtifactManifest struct {
	SchemaVersion    int                     `json:"schemaVersion"`
	Type             string                  `json:"type"`
	PipelineID       uint                    `json:"pipelineId"`
	PipelineRecordID uint                    `json:"pipelineRecordId"`
	Commit           string                  `json:"commit"`
	SourceType       string                  `json:"sourceType,omitempty"`
	SourceID         uint                    `json:"sourceId,omitempty"`
	SourceDigest     string                  `json:"sourceDigest,omitempty"`
	Digest           string                  `json:"digest"`
	SizeBytes        int64                   `json:"sizeBytes,omitempty"`
	Image            *ArtifactImageManifest  `json:"image,omitempty"`
	Archive          *ArtifactFileManifest   `json:"archive,omitempty"`
	Directory        *ArtifactFileManifest   `json:"directory,omitempty"`
	Runtime          ArtifactRuntimeManifest `json:"runtime"`
}

type ArtifactImageManifest struct {
	Tag          string `json:"tag"`
	ID           string `json:"id"`
	RepoDigest   string `json:"repoDigest,omitempty"`
	ImmutableRef string `json:"immutableRef"`
}

type ArtifactFileManifest struct {
	Path string `json:"path"`
}

type ArtifactRuntimeManifest struct {
	Mode         string `json:"mode,omitempty"`
	Port         int    `json:"port,omitempty"`
	StartCommand string `json:"startCommand,omitempty"`
	WorkingDir   string `json:"workingDir,omitempty"`
}
