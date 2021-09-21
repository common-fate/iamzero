package applier

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/common-fate/iamzero/pkg/applier"
	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
)

// CDKFinding is proposed CDK source code changes recommended by IAM Zero
type CDKFinding struct {
	FindingID       string              `json:"findingId"`
	Role            CDKRole             `json:"role"`
	Recommendations []CDKRecommendation `json:"recommendations"`
}

// CDKRole is a reference to a user or role defined in CDK
type CDKRole struct {
	Type    string `json:"type"`
	CDKPath string `json:"cdkPath"`
}

type CDKRecommendation struct {
	Type       string         `json:"type"`
	Statements []CDKStatement `json:"statements"`
}

type CDKStatement struct {
	Resources []CDKResource `json:"resources"`
	Actions   []string      `json:"actions"`
}

type CDKResource struct {
	Reference string  `json:"reference"`
	Type      string  `json:"type"`
	CDKPath   *string `json:"cdkPath,omitempty"`
	ARN       *string `json:"arn,omitempty"`
}

type CDKIAMPolicyApplier struct {
	AWSIAMPolicyApplier applier.AWSIAMPolicyApplier
	Finding             *CDKFinding
	SkipSynth           bool
	CTX                 context.Context
	ApplierBinaryPath   string
	Manifest            string
}

func (t CDKIAMPolicyApplier) GetProjectName() string { return "CDK" }

func (t CDKIAMPolicyApplier) Init() error {

	if !t.SkipSynth {
		fmt.Println("Synthesizing the CDK project with 'cdk synth' so that we can analyse it (you can skip this step by passing the -skip-synth flag)...")

		cmd := exec.CommandContext(t.CTX, "cdk", "synth")
		cmd.Dir = t.AWSIAMPolicyApplier.ProjectPath

		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	// After the stack is synthesized the manifest file will be available
	// at {projectDir}/cdk.out/manifest.json
	t.Manifest = path.Join(t.AWSIAMPolicyApplier.ProjectPath, "cdk.out", "manifest.json")
	t.AWSIAMPolicyApplier.Logger.With("manifest", t.Manifest).Debug("Stack synthesized")

	return nil
}

func (t CDKIAMPolicyApplier) Detect() bool {
	_, errCdk := os.Stat(path.Join(t.AWSIAMPolicyApplier.ProjectPath, "cdk.json"))
	return os.IsExist(errCdk)
}

func (t CDKIAMPolicyApplier) Plan(policy *recommendations.Policy, actions []recommendations.AWSAction) (*applier.PendingChanges, error) {
	t.calculateCDKFinding(policy, actions)
	if t.Finding != nil && t.Finding.Role.CDKPath != "" {
		findingStr, err := json.Marshal(t.Finding)
		if err != nil {
			return nil, err
		}
		t.AWSIAMPolicyApplier.Logger.With("finding", t.Finding.FindingID).Debug("applying finding")

		cmd := exec.CommandContext(t.CTX, t.ApplierBinaryPath, "-f", string(findingStr), "-m", t.Manifest)
		cmd.Stderr = os.Stderr
		stdout, err := cmd.Output()

		if err != nil {
			return nil, err
		}

		var out applier.PendingChanges

		err = json.Unmarshal(stdout, &out)
		if err != nil {
			return nil, err
		}
		t.AWSIAMPolicyApplier.Logger.With("out", out).Debug("parsed applier output")

		return &out, nil
	}
	// not a cdk finding
	return &applier.PendingChanges{}, nil

}

func (t CDKIAMPolicyApplier) Apply(changes *applier.PendingChanges) error {
	for _, o := range *changes {
		err := ioutil.WriteFile(o.Path, []byte(o.Contents), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t CDKIAMPolicyApplier) calculateCDKFinding(policy *recommendations.Policy, actions []recommendations.AWSAction) {
	// only derive a CDK finding if we know that the role that we are
	// giving recommendations for has been defined using CDK
	if policy.Identity.CDKResource == nil {
		return
	}
	f := CDKFinding{
		FindingID: policy.ID,
		Role: CDKRole{
			Type:    policy.Identity.CDKResource.Type,
			CDKPath: policy.Identity.CDKResource.CDKPath,
		},
		Recommendations: []CDKRecommendation{},
	}

	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			rec := CDKRecommendation{
				Type:       "IAMInlinePolicy",
				Statements: []CDKStatement{},
			}
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				// TODO: this should be redesigned to avoid casting from the interface.
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				if ok {
					for _, s := range policy.Statement {
						cdkStatement := CDKStatement{
							Actions: s.Action,
						}
						// TODO: we need to better structure resources so that
						// we have a reference to a CDK resource in an IAM statement
						for _, resource := range alert.Resources {

							var cdkResource CDKResource
							if resource.CDKResource != nil {
								cdkResource = CDKResource{
									Reference: "CDK",
									Type:      resource.CDKResource.Type,
									CDKPath:   &resource.CDKResource.CDKPath,
								}
							} else {
								cdkResource = CDKResource{
									Reference: "IAM",
									ARN:       &resource.ARN,
								}
							}
							cdkStatement.Resources = append(cdkStatement.Resources, cdkResource)
						}
						rec.Statements = append(rec.Statements, cdkStatement)
					}
				}
			}
			f.Recommendations = append(f.Recommendations, rec)
		}
	}

	t.Finding = &f
}
