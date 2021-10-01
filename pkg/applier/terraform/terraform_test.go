package applier_test

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/applier"
	terraformApplier "github.com/common-fate/iamzero/pkg/applier/terraform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestIsAwsIamRole(t *testing.T) {
	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, terraformApplier.IsBlockAwsIamRole(hclfile.Body().Blocks()[3]))
	assert.True(t, terraformApplier.IsBlockAwsIamRole(hclfile.Body().Blocks()[5]))

}

func TestIsInlinePolicy(t *testing.T) {
	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, terraformApplier.IsBlockInlinePolicy(hclfile.Body().Blocks()[3]))
	assert.Len(t, hclfile.Body().Blocks()[5].Body().Blocks(), 1)
	assert.True(t, terraformApplier.IsBlockInlinePolicy(hclfile.Body().Blocks()[5].Body().Blocks()[0]))

}
func TestParseIamBlocks(t *testing.T) {
	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	assert.Len(t, iamBlocks, 1)
	assert.Equal(t, iamBlocks[0], hclfile.Body().Blocks()[5])
}

func TestApplyFindingToBlocks(t *testing.T) {

	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/*"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := fh.OpenFile("./test/example_1/snapshots/snapshot_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	stateFile, _ := fh.OpenStateFile("./test/example_1/terraform.tfstate")
	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: ""}, StateFile: stateFile, Finding: finding}

	stateFileResource, _ := tf.FindResourceInStateFileByArn(finding.Role)
	block := terraformApplier.AwsIamBlock{iamBlocks[0]}
	tf.FileHandler = &terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	err = tf.ApplyFindingToBlock(&block, stateFileResource, hclfile)
	assert.True(t, err == nil)

	assert.Equal(t, string(hclwrite.Format(snapshotFile.Bytes())), string(hclwrite.Format(hclfile.Bytes())))
}

func TestApplyFindingToBlocksWithSpecificBucketResource(t *testing.T) {
	// tests that the terraformApplier correctly adds the join() function in to specify the resource
	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/README.md"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := fh.OpenFile("./test/example_1/snapshots/snapshot_2/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	stateFile, _ := fh.OpenStateFile("./test/example_1/terraform.tfstate")
	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{Logger: &zap.SugaredLogger{},
		ProjectPath: ""}, StateFile: stateFile, Finding: finding}
	stateFileResource, _ := tf.FindResourceInStateFileByArn(finding.Role)
	block := terraformApplier.AwsIamBlock{iamBlocks[0]}
	tf.FileHandler = &terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	err = tf.ApplyFindingToBlock(&block, stateFileResource, hclfile)
	assert.True(t, err == nil)
	assert.Equal(t, string(hclwrite.Format(snapshotFile.Bytes())), string(hclwrite.Format(hclfile.Bytes())))

}

func TestApplyFindingToBlocksV2(t *testing.T) {

	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/*"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	fh := terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	hclfile, err := fh.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := fh.OpenFile("./test/example_1/snapshots/snapshot_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	stateFile, _ := fh.OpenStateFile("./test/example_1/terraform.tfstate")
	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: ""}, StateFile: stateFile, Finding: finding}

	stateFileResource, _ := tf.FindResourceInStateFileByArn(finding.Role)
	block := terraformApplier.AwsIamBlock{iamBlocks[0]}
	tf.FileHandler = &terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	err = tf.ApplyFindingToBlock(&block, stateFileResource, hclfile)
	assert.True(t, err == nil)

	assert.Equal(t, string(hclwrite.Format(snapshotFile.Bytes())), string(hclwrite.Format(hclfile.Bytes())))
}
