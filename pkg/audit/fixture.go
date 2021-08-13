package audit

import (
	"github.com/common-fate/iamzero/pkg/recommendations"
)

// LoadFixture seeds fixture data to be used when testing the auditor functionality
func (a *Auditor) LoadFixture() {
	// a source role which assumes a target role
	source := AWSRole{
		ManagedPolicies: []ManagedPolicy{
			{
				ARN: "arn:aws:iam::111222333444:policy/target-policy",
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
	a.roleStorage.Add(source)

	// a target role which can be assumed from the source role
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
	a.roleStorage.Add(target)

	// an unrelated role which no other role can assume
	unrelated := AWSRole{
		ManagedPolicies: []ManagedPolicy{},
		InlinePolicies:  []InlinePolicy{},
		ARN:             "arn:aws:iam::111222333444:role/unrelated",
		AccountID:       "111222333444",
	}
	a.roleStorage.Add(unrelated)

}
