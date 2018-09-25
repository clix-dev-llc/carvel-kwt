package workspace

import (
	"time"

	"k8s.io/client-go/rest"
)

type Workspace interface {
	Name() string
	CreationTime() time.Time

	Ports() []string

	LastUsedTime() time.Time
	MarkUse() error

	WaitForStart(chan struct{}) error

	Enter() error
	Execute(ExecuteOpts, *rest.Config) error

	Upload(UploadInput, *rest.Config) error // TODO remove rest.Config
	Download(DownloadOutput, *rest.Config) error

	Delete() error
}
