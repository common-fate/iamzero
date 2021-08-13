package audit

import (
	"testing"

	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/stretchr/testify/assert"
)

func TestCanAssume_BasicCase(t *testing.T) {
	source := AWSRole{
		ManagedPolicies: []ManagedPolicy{
			{
				ARN: "arn:aws:iam::111222333444:role/target",
				Document: recommendations.AWSIAMPolicy{
					Version: "2012-10-17",
					Statement: []recommendations.AWSIAMStatement{
						{
							Sid:      "1",
							Effect:   "Allow",
							Action:   []string{"sts:AssumeRole"},
							Resource: []string{"arn:aws:iam::111222333444:role/target"},
						},
					},
				},
			},
		},
		InlinePolicies: []InlinePolicy{},
		ARN:            "arn:aws:iam::123456789012:role/source",
		AccountID:      "123456789012",
	}

	target := AWSRole{
		ManagedPolicies: []ManagedPolicy{},
		InlinePolicies:  []InlinePolicy{},
		ARN:             "arn:aws:iam::111222333444:role/target",
		AccountID:       "111222333444",
		TrustPolicyDocument: TrustPolicyDocument{
			Statement: []recommendations.AWSIAMStatement{
				{
					Effect: "Allow",
					Action: []string{"sts:AssumeRole"},
					Principal: &recommendations.AWSIAMPrincipal{
						AWS: "arn:aws:iam::123456789012:role/source",
					},
				},
			},
		},
	}

	canAssume := source.CanAssume(target)
	assert.True(t, canAssume)
}

func TestCannotAssumeIfTrustPolicyNotConfigured(t *testing.T) {
	source := AWSRole{
		ManagedPolicies: []ManagedPolicy{
			{
				ARN: "arn:aws:iam::111222333444:policy/source-policy",
				Document: recommendations.AWSIAMPolicy{
					Version: "2012-10-17",
					Statement: []recommendations.AWSIAMStatement{
						{
							Sid:      "1",
							Effect:   "Allow",
							Action:   []string{"sts:AssumeRole"},
							Resource: []string{"arn:aws:iam::111222333444:role/target"},
						},
					},
				},
			},
		},
		InlinePolicies: []InlinePolicy{},
		ARN:            "arn:aws:iam::123456789012:role/source",
		AccountID:      "123456789012",
	}

	// target has no trust policy document
	target := AWSRole{
		ManagedPolicies: []ManagedPolicy{},
		InlinePolicies:  []InlinePolicy{},
		ARN:             "arn:aws:iam::111222333444:role/target",
		AccountID:       "111222333444",
	}

	canAssume := source.CanAssume(target)
	assert.False(t, canAssume)
}
