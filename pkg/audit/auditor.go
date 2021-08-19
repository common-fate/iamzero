package audit

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/common-fate/iamzero/pkg/policies"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// ManagedPolicy is a managed IAM policy
type ManagedPolicy struct {
	ARN      string                `json:"arn"`
	Document policies.AWSIAMPolicy `json:"document"`
}

// InlinePolicy is an inline IAM policy
type InlinePolicy struct {
	Name     string                `json:"name"`
	Document policies.AWSIAMPolicy `json:"document"`
}

type auditRoles []string

func (i *auditRoles) String() string {
	return strings.Join(*i, ",")
}

func (i *auditRoles) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// AWSRoleStorage stores AWSRoles in memory
// and is goroutine-safe
type AWSRoleStorage struct {
	sync.Mutex
	iamRoles []AWSRole
}

func NewAWSRoleStorage() *AWSRoleStorage {
	return &AWSRoleStorage{
		iamRoles: []AWSRole{},
	}
}

func (s *AWSRoleStorage) Add(r AWSRole) {
	s.Lock()
	defer s.Unlock()
	s.iamRoles = append(s.iamRoles, r)
}

func (s *AWSRoleStorage) List() []AWSRole {
	return s.iamRoles
}

// CDKResourceStorage stores CDKResources in memory
// and is goroutine-safe
type CDKResourceStorage struct {
	sync.Mutex
	resources []policies.CDKResource
}

func NewCDKResourceStorage() *CDKResourceStorage {
	return &CDKResourceStorage{
		resources: []policies.CDKResource{},
	}
}

func (s *CDKResourceStorage) Add(r policies.CDKResource) {
	s.Lock()
	defer s.Unlock()
	s.resources = append(s.resources, r)
}

// GetByPhysicalID looks up a CDK resource by it's Physical ID
func (s *CDKResourceStorage) GetByPhysicalID(id string) *policies.CDKResource {
	for _, r := range s.resources {
		if r.PhysicalID == id {
			return &r
		}
	}
	return nil
}

// Auditor reads resources across a cloud environment to give an
// understanding of what is deployed, and where.
//
// The auditor helps IAM Zero solve complex IAM use cases
// such as implementing cross-account role access.
type Auditor struct {
	// the list of roles to assume for auditing
	// each AWS account will have a role
	// the profile that IAM Zero runs as must have permission
	// to assume each of these roles
	auditRoles auditRoles

	// a map of policy ARN to the actual policy document
	policyMap sync.Map

	// all IAM roles across all accounts audited
	roleStorage  *AWSRoleStorage
	links        []AssumeRoleLink
	cdkResources *CDKResourceStorage

	log *zap.SugaredLogger
}

func New() *Auditor {
	return &Auditor{
		roleStorage:  NewAWSRoleStorage(),
		cdkResources: NewCDKResourceStorage(),
	}
}

func (a *Auditor) AddFlags(fs *flag.FlagSet) {
	fs.Var(&a.auditRoles, "audit-role", "an audit role ARN to assume for auditing (multiple arguments allowed)")
}

// Setup configures logging for the auditor
func (a *Auditor) Setup(log *zap.SugaredLogger) {
	a.log = log
}

// LoadResources loads all resources across all accounts into
// a cache.
//
// Note: our initial use case loads IAM roles only
// to map cross-account role access
func (a *Auditor) LoadResources(ctx context.Context) error {
	if len(a.auditRoles) == 0 {
		return errors.New("no audit roles supplied")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return err
	}
	svc := sts.NewFromConfig(cfg)

	for _, role := range a.auditRoles {
		a.log.With("role", role).Info("assuming role for audit")
		creds := stscreds.NewAssumeRoleProvider(svc, role)
		cfg.Credentials = aws.NewCredentialsCache(creds)

		// Create service client value configured for credentials
		// from assumed role.
		client := iam.NewFromConfig(cfg)

		policies, err := client.ListPolicies(ctx, &iam.ListPoliciesInput{})
		if err != nil {
			return err
		}

		g, gctx := errgroup.WithContext(ctx)

		for _, p := range policies.Policies {
			policy := p
			g.Go(func() error { return a.fetchPolicyDetails(gctx, client, policy) })
		}

		err = g.Wait()
		if err != nil {
			return err
		}

		// TODO: handle paginated response
		roles, err := client.ListRoles(ctx, &iam.ListRolesInput{})
		if err != nil {
			return err
		}

		g, gctx = errgroup.WithContext(ctx)

		for _, r := range roles.Roles {
			role := r
			g.Go(func() error { return a.FetchDetailsForRole(gctx, client, role) })
		}
		err = g.Wait()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Auditor) fetchPolicyDetails(ctx context.Context, client *iam.Client, p types.Policy) error {
	a.log.With("policy", p.Arn).Debug("fetching policy document")
	details, err := client.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: p.Arn,
		VersionId: p.DefaultVersionId,
	})
	if err != nil {
		return err
	}
	var doc policies.AWSIAMPolicy
	s, err := url.QueryUnescape(*details.PolicyVersion.Document)
	if err != nil {
		return err
	}
	a.log.With("policy", p.Arn, "document", s).Debug("unmarshalling policy document")

	err = json.Unmarshal([]byte(s), &doc)
	if err != nil {
		return err
	}
	a.policyMap.Store(*p.Arn, doc)
	return nil
}

func (a *Auditor) FetchDetailsForRole(ctx context.Context, client *iam.Client, r types.Role) error {
	arnParsed, err := arn.Parse(*r.Arn)
	if err != nil {
		return err
	}
	var doc TrustPolicyDocument
	s, err := url.QueryUnescape(*r.AssumeRolePolicyDocument)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(s), &doc)
	if err != nil {
		return err
	}
	role := AWSRole{
		ARN:                 *r.Arn,
		AccountID:           arnParsed.AccountID,
		TrustPolicyDocument: doc,
	}

	a.log.With("role", r.Arn).Debug("fetching attached managed policies")

	managedPolicies, err := client.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
		RoleName: r.RoleName,
	})
	if err != nil {
		return err
	}

	for _, p := range managedPolicies.AttachedPolicies {
		arnParsed, err = arn.Parse(*p.PolicyArn)
		if err != nil {
			return err
		}
		policy, found := a.policyMap.Load(*p.PolicyArn)
		if found {
			role.ManagedPolicies = append(role.ManagedPolicies, ManagedPolicy{
				ARN:      *p.PolicyArn,
				Document: policy.(policies.AWSIAMPolicy),
			})
		}
	}

	a.log.With("role", r.Arn).Debug("fetching inline policies")

	inlinePolicies, err := client.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
		RoleName: r.RoleName,
	})
	if err != nil {
		return err
	}

	for _, p := range inlinePolicies.PolicyNames {
		a.log.With("role", r.Arn, "policy-name", p).Debug("fetching policy document")
		details, err := client.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
			PolicyName: &p,
			RoleName:   r.RoleName,
		})
		if err != nil {
			return err
		}
		var doc policies.AWSIAMPolicy
		s, err := url.QueryUnescape(*details.PolicyDocument)
		if err != nil {
			return err
		}

		err = json.Unmarshal([]byte(s), &doc)
		if err != nil {
			return err
		}
		role.InlinePolicies = append(role.InlinePolicies, InlinePolicy{
			Name:     p,
			Document: doc,
		})
	}
	a.roleStorage.Add(role)
	return nil
}

// GetRoles returns the cached IAM roles across all AWS accounts
func (a *Auditor) GetRoles() []AWSRole {
	return a.roleStorage.List()
}

func (a *Auditor) GetLinks() []AssumeRoleLink {
	return a.links
}

func (a *Auditor) GetCDKResourceByPhysicalID(id string) *policies.CDKResource {
	return a.cdkResources.GetByPhysicalID(id)
}

// GetPhysicalIDFromARNResource transforms an ARN resource string into
// a CloudFormation physical ID.
// We need to use this because when we parse the ARN read by IAM Zero
// it looks something like `role/my-example-role` - whereas the physical
// ID stored in CloudFormation would be `my-example-role`.
// This function extracts the physical ID by removing the
// `role/` at the beginning of the string.
func GetPhysicalIDFromARNResource(resource string) (string, error) {
	split := strings.Split(resource, "/")
	if len(split) < 2 {
		return "", fmt.Errorf("could not split %s into a Physical ID", resource)
	}
	return split[1], nil
}
