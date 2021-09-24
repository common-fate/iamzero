package applier

import (
	"fmt"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"go.uber.org/zap"
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
	// This method will be called before running the plan or apply steps
	Init() error
	// The Detect function shoudl evaluate whether there is a project matching this applier at the project path
	Detect() bool
	// Plan will detect the changes required to apply a policy and return the results
	Plan(*recommendations.Policy, []recommendations.AWSAction) (*PendingChanges, error)
	// Apply will write the changes to the project files
	Apply(*PendingChanges) error
	// This should return a string descripting what type of project this applier operates on e.g "Terraform" or "CDK"
	GetProjectName() string
}

type PolicyAppliers []PolicyApplier
type AWSIAMPolicyApplier struct {
	// Policy      recommendations.Policy
	// Actions     []recommendations.AWSAction
	ProjectPath string
	Logger      *zap.SugaredLogger
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
