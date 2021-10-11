package applier_test

import (
	"path"
	"testing"

	"github.com/common-fate/iamzero/pkg/applier"
	terraformApplier "github.com/common-fate/iamzero/pkg/applier/terraform"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
)

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

func TestApplyFindingToBlocks(t *testing.T) {

	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/*"
	finding := &terraformApplier.TerraformFinding{FindingId: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: "./test/example_1/"}, Finding: finding}
	err := tf.Init()
	if err != nil {
		t.Fatal(err)
	}

	_, err = tf.Plan()
	assert.True(t, err == nil)

	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_1/", "main.tf")
}

func TestApplyFindingToBlocksWithSpecificBucketResource(t *testing.T) {
	// tests that the terraformApplier correctly adds the join() function in to specify the resource
	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/README.md"
	finding := &terraformApplier.TerraformFinding{FindingId: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: "./test/example_1/"}, Finding: finding}
	err := tf.Init()
	if err != nil {
		t.Fatal(err)
	}

	_, err = tf.Plan()
	assert.True(t, err == nil)
	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_2/", "main.tf")

}

func TestApplyFindingForMultiFile(t *testing.T) {
	// tests that the terraformApplier correctly adds the join() function in to specify the resource
	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role-pa"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket4/*"
	finding := &terraformApplier.TerraformFinding{FindingId: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: "./test/example_1/"}, Finding: finding}
	err := tf.Init()
	if err != nil {
		t.Fatal(err)
	}
	_, err = tf.Plan()
	if err != nil {
		t.Fatal(err)
	}

	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_3/", "main.tf")
	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_3/", "/modules/ec2/main.tf")
	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_3/", "/modules/s3/main.tf")
	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_3/", "/modules/ec2/variables.tf")
	AssertFilesEqual(t, tf.FileHandler, "./test/example_1/", "./test/example_1/snapshots/snapshot_3/", "/modules/s3/outputs.tf")

}

func AssertFilesEqual(t *testing.T, fh *terraformApplier.FileHandler, orginalPath string, snapshotPath string, filePath string) {
	original, err := fh.OpenFile(path.Join(orginalPath, filePath), false)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fh.OpenFile(path.Join(snapshotPath, filePath), false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(hclwrite.Format(snapshot.Bytes())), string(hclwrite.Format(original.Bytes())))
}
