package applier

import (
	"fmt"

	"github.com/common-fate/iamzero/pkg/recommendations"
)

// ApplierOutput is the output from the iamzero-applier command
type PendingChanges []PendingChange

// ChangedFile is a file change made by the applier to a specific source code file.
type PendingChange struct {
	// the path of the file which has been modified
	Path string `json:"path"`
	// the new contents of the file
	Contents string `json:"contents"`
}

type PolicyApplier interface {
	Init() error
	Detect() bool
	Plan() (PendingChanges, error)
	Apply(PendingChanges) error
	GetProjectName() string
}

type PolicyAppliers []PolicyApplier
type AWSIAMPolicyApplier struct {
	Policy      recommendations.Policy
	Actions     []recommendations.AWSAction
	ProjectPath string
}

func (changes PendingChanges) RenderDiff() error {
	for _, change := range changes {
		// @TODO Changes for change.FilePath may want to make this message nicer
		fmt.Printf("Changes for the following file (%s)", change.Path)
		diff, err := GetDiff(change.Path, string(change.Contents), true)
		if err != nil {
			return err
		}
		fmt.Println(diff)
	}
	return nil
}
