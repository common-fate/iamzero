package applier_test

import (
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
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: "./test/example_1/"}, Finding: finding}
	err := tf.Init()
	if err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := tf.FileHandler.OpenFile("./test/example_1/snapshots/snapshot_1/main.tf", false)

	if err != nil {
		t.Fatal(err)
	}
	hclfile, err := tf.FileHandler.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	block := tf.Blocks.GetBlock(tf.StateFileResources.Get(iamRoleARN).Key)
	err = tf.ApplyFindingToBlock(block)
	assert.True(t, err == nil)

	assert.Equal(t, string(hclwrite.Format(snapshotFile.Bytes())), string(hclwrite.Format(hclfile.Bytes())))
}

func TestApplyFindingToBlocksWithSpecificBucketResource(t *testing.T) {
	// tests that the terraformApplier correctly adds the join() function in to specify the resource
	iamRoleARN := "arn:aws:iam::12345678910:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket3/README.md"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: "./test/example_1/"}, Finding: finding}
	err := tf.Init()
	if err != nil {
		t.Fatal(err)
	}
	snapshotFile, err := tf.FileHandler.OpenFile("./test/example_1/snapshots/snapshot_2/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	hclfile, err := tf.FileHandler.OpenFile("./test/example_1/main.tf", false)
	if err != nil {
		t.Fatal(err)
	}
	block := tf.Blocks.GetBlock(tf.StateFileResources.Get(iamRoleARN).Key)
	err = tf.ApplyFindingToBlock(block)
	assert.True(t, err == nil)
	assert.Equal(t, string(hclwrite.Format(snapshotFile.Bytes())), string(hclwrite.Format(hclfile.Bytes())))

}
