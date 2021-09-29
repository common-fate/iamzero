package applier_test

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/applier"
	terraformApplier "github.com/common-fate/iamzero/pkg/applier/terraform"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var initial = []byte(`terraform {
	required_providers {
	  aws = {
		source  = "hashicorp/aws"
		version = "~> 3.27"
	  }
	}
  
	required_version = ">= 0.14.9"
  }
  
  provider "aws" {
	profile = "default"
	region  = "ap-southeast-2"
  }
  
  resource "aws_s3_bucket" "iamzero-tf-example-bucket" {
	bucket = "iamzero-tf-example-bucket"
	acl    = "private"
  }
  
  locals {
	#This should the the role of the AWS account that the user is using to login
	aws-user-arn = "arn:aws:iam::312231318920:root"
	role-name = "iamzero-tf-overprivileged-role"
  }
  
  resource "aws_iam_role" "iamzero-overprivileged-role" {
	name = local.name //"iamzero-tf-overprivileged-role"
	assume_role_policy = jsonencode({
	  Version = "2012-10-17"
	  Statement = [
		{
		  Action = "sts:AssumeRole"
		  Effect = "Allow"
		  Sid    = ""
		  Principal = {
			AWS = local.aws-user-arn
		  }
		},
	  ]
	})
	inline_policy {
	  policy = jsonencode({
		Version = "2012-10-17"
		Statement = [
		  {
			Action   = ["s3:GetObject"]
			Effect   = "Allow"
			Resource = aws_s3_bucket.iamzero-tf-example-bucket.arn
		  },
		]
	  })
	  name = "iamzero-generated-iam-policy-0"
	}
  }`)

var snapshot = []byte(`terraform {
	required_providers {
	  aws = {
		source  = "hashicorp/aws"
		version = "~> 3.27"
	  }
	}
  
	required_version = ">= 0.14.9"
  }
  
  provider "aws" {
	profile = "default"
	region  = "ap-southeast-2"
  }
  
  resource "aws_s3_bucket" "iamzero-tf-example-bucket" {
	bucket = "iamzero-tf-example-bucket"
	acl    = "private"
  }
  
  locals {
	#This should the the role of the AWS account that the user is using to login
	aws-user-arn = "arn:aws:iam::312231318920:root"
	role-name = "iamzero-tf-overprivileged-role"
  }
  
  resource "aws_iam_role" "iamzero-overprivileged-role" {
	name = local.name //"iamzero-tf-overprivileged-role"
	assume_role_policy = jsonencode({
	  Version = "2012-10-17"
	  Statement = [
		{
		  Action = "sts:AssumeRole"
		  Effect = "Allow"
		  Sid    = ""
		  Principal = {
			AWS = local.aws-user-arn
		  }
		},
	  ]
	})
	inline_policy {
	  policy = jsonencode({
		Version = "2012-10-17"
		Statement = [
		  {
			Action   = ["s3:GetObject"]
			Effect   = "Allow"
			Resource = aws_s3_bucket.iamzero-tf-example-bucket.arn
		  },
		]
	  })
	  name = "iamzero-generated-iam-policy-0"
	}
  }`)

var snapshotSpecificResource = []byte(`terraform {
	required_providers {
	  aws = {
		source  = "hashicorp/aws"
		version = "~> 3.27"
	  }
	}
  
	required_version = ">= 0.14.9"
  }
  
  provider "aws" {
	profile = "default"
	region  = "ap-southeast-2"
  }
  
  resource "aws_s3_bucket" "iamzero-tf-example-bucket" {
	bucket = "iamzero-tf-example-bucket"
	acl    = "private"
  }
  
  locals {
	#This should the the role of the AWS account that the user is using to login
	aws-user-arn = "arn:aws:iam::312231318920:root"
	role-name = "iamzero-tf-overprivileged-role"
  }
  
  resource "aws_iam_role" "iamzero-overprivileged-role" {
	name = local.name //"iamzero-tf-overprivileged-role"
	assume_role_policy = jsonencode({
	  Version = "2012-10-17"
	  Statement = [
		{
		  Action = "sts:AssumeRole"
		  Effect = "Allow"
		  Sid    = ""
		  Principal = {
			AWS = local.aws-user-arn
		  }
		},
	  ]
	})
	inline_policy {
	  policy = jsonencode({
		Version = "2012-10-17"
		Statement = [
		  {
			Action   = ["s3:GetObject"]
			Effect   = "Allow"
			Resource = join("/", [aws_s3_bucket.iamzero-tf-example-bucket.arn,"README.md"])
		  },
		]
	  })
	  name = "iamzero-generated-iam-policy-0"
	}
  }`)

var terraformShow = []byte(`{"format_version":"0.1","terraform_version":"0.14.9","values":{"root_module":{"resources":[{"address":"aws_iam_role.iamzero-overprivileged-role","mode":"managed","type":"aws_iam_role","name":"iamzero-overprivileged-role","provider_name":"registry.terraform.io/hashicorp/aws","schema_version":0,"values":{"arn":"arn:aws:iam::312231318920:role/iamzero-tf-overprivileged-role","assume_role_policy":"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"\",\"Effect\":\"Allow\",\"Principal\":{\"AWS\":\"arn:aws:iam::312231318920:root\"},\"Action\":\"sts:AssumeRole\"}]}","create_date":"2021-09-03T03:30:22Z","description":"","force_detach_policies":false,"id":"iamzero-tf-overprivileged-role","inline_policy":[{"name":"tf-example-policy","policy":"{\"Statement\":[{\"Action\":[\"*\"],\"Effect\":\"Allow\",\"Resource\":\"*\"}],\"Version\":\"2012-10-17\"}"}],"managed_policy_arns":[],"max_session_duration":3600,"name":"iamzero-tf-overprivileged-role","name_prefix":null,"path":"/","permissions_boundary":null,"tags":{},"tags_all":{},"unique_id":"AROAURMTP2WECJCRJBHTS"}},{"address":"aws_s3_bucket.iamzero-tf-example-bucket","mode":"managed","type":"aws_s3_bucket","name":"iamzero-tf-example-bucket","provider_name":"registry.terraform.io/hashicorp/aws","schema_version":0,"values":{"acceleration_status":"","acl":"private","arn":"arn:aws:s3:::iamzero-tf-example-bucket","bucket":"iamzero-tf-example-bucket","bucket_domain_name":"iamzero-tf-example-bucket.s3.amazonaws.com","bucket_prefix":null,"bucket_regional_domain_name":"iamzero-tf-example-bucket.s3.ap-southeast-2.amazonaws.com","cors_rule":[],"force_destroy":false,"grant":[],"hosted_zone_id":"Z1WCIGYICN2BYD","id":"iamzero-tf-example-bucket","lifecycle_rule":[],"logging":[],"object_lock_configuration":[],"policy":null,"region":"ap-southeast-2","replication_configuration":[],"request_payer":"BucketOwner","server_side_encryption_configuration":[],"tags":{},"tags_all":{},"versioning":[{"enabled":false,"mfa_delete":false}],"website":[],"website_domain":null,"website_endpoint":null}}]}}}`)

func TestIsAwsIamRole(t *testing.T) {
	hclfile, err := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, terraformApplier.IsBlockAwsIamRole(hclfile.Body().Blocks()[3]))
	assert.True(t, terraformApplier.IsBlockAwsIamRole(hclfile.Body().Blocks()[4]))

}

func TestIsInlinePolicy(t *testing.T) {
	hclfile, err := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, terraformApplier.IsBlockInlinePolicy(hclfile.Body().Blocks()[3]))
	assert.Len(t, hclfile.Body().Blocks()[4].Body().Blocks(), 1)
	assert.True(t, terraformApplier.IsBlockInlinePolicy(hclfile.Body().Blocks()[4].Body().Blocks()[0]))

}
func TestParseIamBlocks(t *testing.T) {
	hclfile, err := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	assert.Len(t, iamBlocks, 1)
	assert.Equal(t, iamBlocks[0], hclfile.Body().Blocks()[4])

}

func TestApplyFindingToBlocks(t *testing.T) {

	iamRoleARN := "arn:aws:iam::312231318920:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket/*"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	hclfile, diag := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if diag != nil {
		t.Fatal(diag)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	stateFile, _ := terraformApplier.MarshalStateFileToGo(terraformShow)
	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{
		ProjectPath: ""}, StateFile: stateFile, Finding: finding}

	stateFileResource, _ := tf.FindResourceInStateFileByArn(finding.Role)
	block := terraformApplier.AwsIamBlock{iamBlocks[0]}
	tf.FileHandler = &terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	err := tf.ApplyFindingToBlock(&block, stateFileResource, hclfile)
	assert.True(t, err == nil)

	assert.Equal(t, string(hclwrite.Format(snapshot)), string(hclwrite.Format(hclfile.Bytes())))

}
func TestStringCompareAttributeValue(t *testing.T) {
	hclfile, err := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if err != nil {
		t.Fatal(err)
	}
	at := hclfile.Body().Blocks()[4].Body().Attributes()["name"]
	assert.True(t, terraformApplier.StringCompareAttributeValue(at, "local.name"))
	at = hclfile.Body().Blocks()[3].Body().Attributes()["role-name"]
	assert.True(t, terraformApplier.StringCompareAttributeValue(at, "iamzero-tf-overprivileged-role"))

}

func TestApplyFindingToBlocksWithSpecificBucketResource(t *testing.T) {
	// tests that the terraformApplier correctly adds the join() function in to specify the resource
	iamRoleARN := "arn:aws:iam::312231318920:role/iamzero-tf-overprivileged-role"
	actionsDemo := []string{"s3:GetObject"}
	bucketArn := "arn:aws:s3:::iamzero-tf-example-bucket/README.md"
	finding := &terraformApplier.TerraformFinding{FindingID: "abcde", Role: iamRoleARN, Recommendations: []terraformApplier.TerraformRecommendation{{Type: "IAMInlinePolicy", Statements: []terraformApplier.TerraformStatement{{Resources: []terraformApplier.TerraformResource{{Reference: bucketArn, ARN: &bucketArn}}, Actions: actionsDemo}}}}}

	hclfile, diag := hclwrite.ParseConfig(initial, "./", hcl.InitialPos)
	if diag != nil {
		t.Fatal(diag)
	}
	iamBlocks := terraformApplier.ParseHclFileForAwsIamBlocks(hclfile)
	stateFile, _ := terraformApplier.MarshalStateFileToGo(terraformShow)
	tf := terraformApplier.TerraformIAMPolicyApplier{AWSIAMPolicyApplier: applier.AWSIAMPolicyApplier{Logger: &zap.SugaredLogger{},
		ProjectPath: ""}, StateFile: stateFile, Finding: finding}
	stateFileResource, _ := tf.FindResourceInStateFileByArn(finding.Role)
	block := terraformApplier.AwsIamBlock{iamBlocks[0]}
	tf.FileHandler = &terraformApplier.FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	err := tf.ApplyFindingToBlock(&block, stateFileResource, hclfile)
	assert.True(t, err == nil)
	assert.Equal(t, string(hclwrite.Format(snapshotSpecificResource)), string(hclwrite.Format(hclfile.Bytes())))

}
