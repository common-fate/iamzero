package recommendations

import "github.com/common-fate/iamzero/pkg/audit"

func NewAdvisor(auditor *audit.Auditor) *Advisor {
	return &Advisor{
		auditor: auditor,
		AlertsMapping: map[string][]AdvisoryTemplate{
			"dynamodb:GetItem": {
				AdvisoryTemplate{
					Policy: []Statement{
						{
							Action: []string{
								"dynamodb:GetItem",
								"dynamodb:BatchGetItem",
								"dynamodb:Scan",
								"dynamodb:Query",
								"dynamodb:ConditionCheckItem",
							},
							Resource: []string{"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}"},
						},
					},
					Comment: "Allow all read-only actions on the table",
					DocLink: "https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/read-only-permissions-on-table-items.html",
				},
				AdvisoryTemplate{
					Policy: []Statement{
						{
							Action: []string{
								"dynamodb:GetShardIterator",
								"dynamodb:Scan",
								"dynamodb:Query",
								"dynamodb:DescribeStream",
								"dynamodb:GetRecords",
								"dynamodb:ListStreams",
							},
							Resource: []string{
								"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}/index/*",
								"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}/stream/*",
							},
						},
						{
							Action: []string{
								"dynamodb:BatchGetItem",
								"dynamodb:BatchWriteItem",
								"dynamodb:ConditionCheckItem",
								"dynamodb:PutItem",
								"dynamodb:DescribeTable",
								"dynamodb:DeleteItem",
								"dynamodb:GetItem",
								"dynamodb:Scan",
								"dynamodb:Query",
								"dynamodb:UpdateItem",
							},
							Resource: []string{
								"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}",
							},
						},
						{
							Action: []string{
								"dynamodb:DescribeLimits",
							},
							Resource: []string{
								"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}",
								"arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}/index/*",
							},
						},
					},
					Comment: "Allow read and write operations on the table",
					DocLink: "https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/iam-policy-example-data-crud.html",
				},
			},
			"s3:PutObject": {
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:PutObject"},
						Resource: []string{"arn:aws:s3:::{{ .Bucket }}"},
					}},
					Comment: "Allow PutObject access to the bucket",
				},
			},
			"s3:CreateBucket": {
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:CreateBucket"},
						Resource: []string{"arn:aws:s3:::{{ .Bucket }}"},
					}},
					Comment: "Allow creating the specific bucket",
				},
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:CreateBucket"},
						Resource: []string{"arn:aws:s3:::*"},
					}},
					Comment: "Allow creating all buckets",
				},
			},
			"s3:DeleteBucket": {
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:DeleteBucket"},
						Resource: []string{"arn:aws:s3:::{{ .Bucket }}"},
					}},
					Comment: "Allow deleting the specific bucket",
				},
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:DeleteBucket"},
						Resource: []string{"arn:aws:s3:::*"},
					}},
					Comment: "Allow deleting all buckets",
				},
			},
			"s3:HeadObject": {
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:GetObject"},
						Resource: []string{"arn:aws:s3:::{{ .Bucket }}/*"},
					}},
					Comment: "Allow access to the bucket",
				},
			},
			"s3:ListBuckets": {
				AdvisoryTemplate{
					Policy: []Statement{{
						Action:   []string{"s3:ListBucket"},
						Resource: []string{"arn:aws:s3:::*"},
					}},
					Comment: "Allows ListObject access to all buckets",
				},
			},
		},
	}
}

// Advise
func (a *Advisor) Advise(e AWSEvent) ([]*JSONAdvice, error) {
	key := e.Data.Service + ":" + e.Data.Operation

	advisoryTemplates := a.AlertsMapping[key]
	var advices []*JSONAdvice

	for _, advisoryTemplate := range advisoryTemplates {
		advice, err := a.CreateAdviceFromEvent(&e, advisoryTemplate)
		if err != nil {
			return nil, err
		}

		advices = append(advices, advice)
	}
	return advices, nil
}
