terraform {
  backend "s3" {
    bucket = "tf-remote-state-demo-bucket"
    key    = "path/to/my/key"
    region = "ap-southeast-2"
  }
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

# S3 state bucket for terraform
resource "aws_s3_bucket" "iamzero-tf-example-bucket3" {
  bucket = "iamzero-tf-example-bucket3"
  acl    = "private"
}
resource "aws_s3_bucket" "tf-remote-state-demo-bucket" {
  bucket = "tf-remote-state-demo-bucket"
  acl    = "private"
}

locals {
  #This should the the role of the AWS account that the user is using to login
  aws-user-arn = "arn:aws:iam::12345678910:root"
}

resource "aws_iam_role" "iamzero-overprivileged-role" {
  name = "iamzero-tf-overprivileged-role"
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
          Resource = aws_s3_bucket.iamzero-tf-example-bucket3.arn
        },
      ]
    })
    name = "iamzero-generated-iam-policy-0"
  }
}

module "ec2"{
  source = "./modules/ec2/"
}

module "s3"{
  source = "./modules/s3/"
}