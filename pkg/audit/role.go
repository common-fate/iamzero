package audit

import (
	"github.com/common-fate/iamzero/pkg/policies"
)

// AWSRole is a role
type AWSRole struct {
	ARN                 string              `json:"arn"`
	AccountID           string              `json:"accountId"`
	ManagedPolicies     []ManagedPolicy     `json:"managedPolicies"`
	InlinePolicies      []InlinePolicy      `json:"inlinePolicies"`
	TrustPolicyDocument TrustPolicyDocument `json:"trustPolicyDocument"`
}

// TrustPolicyDocument describes the trust relationship
// allowing other entities to assume the role
// NOTE: our initial implementation doesn't handle
// AWS Services with trust relationships and is likely to
// panic trying to deserialize these.
// This is because the Principal field is an object with a
// Service key, not an "AWS" key as is used when setting up
// trusts with user-managed roles.
type TrustPolicyDocument struct {
	Version   string
	Statement []policies.AWSIAMStatement
}

// CanAssume tests whether a role can assume another
// It checks whether there are any
//
// NOTE: our initial implementation for this is naive
// and doesn't consider Deny statements in the role policies
//
// To determine whether a source role can assume a target role,
// we run the following process
//
// 1. Check if the source role has an `sts:AssumeRole` statement
// allowing access to the target role
//
// 2. Check if the target role has a trust relationship allowing
// the source role to assume it
func (a *AWSRole) CanAssume(target AWSRole) bool {
	hasPolicy := a.hasPolicyToAssumeTarget(target)
	targetRoleHasTrustRelationship := target.HasTrustRelationshipAllowingSourceAssumption(*a)

	return hasPolicy && targetRoleHasTrustRelationship
}

// checks that the role has the correct policy applied to
// assume the target role
// NOTE: doesn't check if there are conflicting DENY statements
func (a *AWSRole) hasPolicyToAssumeTarget(target AWSRole) bool {
	statements := []policies.AWSIAMStatement{}

	for _, p := range a.ManagedPolicies {
		statements = append(statements, p.Document.Statement...)
	}
	for _, p := range a.InlinePolicies {
		statements = append(statements, p.Document.Statement...)
	}

	for _, s := range statements {
		if statementAllowsTargetRoleAssumption(s, target) {
			return true
		}
	}
	return false
}

// HasTrustRelationshipAllowingSourceAssumption checks whether the trust relationship
// allows a source role to assume `a`
//
// NOTE: doesn't handle conflicting DENY statements
func (a *AWSRole) HasTrustRelationshipAllowingSourceAssumption(source AWSRole) bool {
	for _, s := range a.TrustPolicyDocument.Statement {
		if trustPolicyStatementAllowsSourceRoleAssumption(s, source) {
			return true
		}
	}
	return false
}

// NOTE: doesn't handle wildcards
func trustPolicyStatementAllowsSourceRoleAssumption(s policies.AWSIAMStatement, source AWSRole) bool {
	includesAssumeRole := false
	for _, a := range s.Action {
		if a == "sts:AssumeRole" {
			includesAssumeRole = true
		}
	}
	if !includesAssumeRole {
		return false
	}

	// if the source role is included in the principal assumption will be allowed
	return s.Principal.AWS == source.ARN
}

// NOTE: doesn't handle wildcards (in either the resource nor the action)
func statementAllowsTargetRoleAssumption(s policies.AWSIAMStatement, target AWSRole) bool {
	// whether sts:AssumeRole is included as an allowed action
	includesAssumeRole := false
	for _, a := range s.Action {
		if a == "sts:AssumeRole" {
			includesAssumeRole = true
		}
	}
	if !includesAssumeRole {
		return false
	}

	// whether the target role ARN is included as a resource
	includesTargetResource := false
	for _, r := range s.Resource {
		if r == target.ARN {
			includesTargetResource = true
		}
	}

	// if includesTargetResource is true, both the action and the resource are
	// allowed, so the statement allows the role to be assumed!
	return includesTargetResource
}
