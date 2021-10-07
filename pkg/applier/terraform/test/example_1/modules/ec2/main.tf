

provider "aws" {
  profile = "default"
  region  = "ap-southeast-2"
}
locals {
  #This should the the role of the AWS account that the user is using to login
  aws-user-arn = "arn:aws:iam::12345678910:root"
}
# Policy attachment 

data "aws_iam_policy_document" "iamzero-overprivileged-role-policy-data" {
  statement {

    actions = [
      "*"
    ]

    resources = ["*"]
  }
}

resource "aws_iam_policy" "iamzero-overprivileged-role-policy" {
  name   = "iamzero-overprivileged-role-policy"
  path   = "/"
  policy = data.aws_iam_policy_document.iamzero-overprivileged-role-policy-data.json
}

resource "aws_iam_policy_attachment" "iamzero-overprivileged-role-policy-attachment" {
  name       = "iamzero-overprivileged-role-policy-attachment"
  roles      = [aws_iam_role.iamzero-overprivileged-role-pa.name]
  policy_arn = aws_iam_policy.iamzero-overprivileged-role-policy.arn
}

resource "aws_iam_role" "iamzero-overprivileged-role-pa" {
  name = "iamzero-tf-overprivileged-role-pa"
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
  

  
}