package recommendations

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"text/template/parse"

	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/google/uuid"
)

type AdvisoryTemplate struct {
	Policy  []Statement
	Comment string
	DocLink string
}

type Statement struct {
	Action   []string
	Resource []string
}

type LeastPrivilegePolicy struct {
	ID        string
	AWSPolicy policies.AWSIAMPolicy
	Comment   string
	RoleName  string
	Resources []CloudResourceInstance
}

// CreateAdviceFromEvent runs the received event through the AdvisoryTemplate to generate a least-privilege
// advice
func (a *Advisor) CreateAdviceFromEvent(e *AWSEvent, r AdvisoryTemplate) (*LeastPrivilegePolicy, error) {
	var iamStatements []policies.AWSIAMStatement
	resources := []CloudResourceInstance{}

	// extract variables from the API call to insert in our recommended policies
	vars := e.Data.Parameters

	// include the region and account as variables available for templating
	// TODO: need to test this and consider edge cases where Parameters could contain a separate Region variable!
	vars["Region"] = e.Data.Region
	vars["Account"] = e.Identity.Account

	// generate AWS statements for each template statement we have
	for _, statement := range r.Policy {
		// the actual ARN of the resource after executing the template
		// for example, `arn:aws:s3:::test-bucket/test-object`
		renderedResources := []string{}

		for _, resourceTemplate := range statement.Resource {
			// template out each resource
			tmpl, err := template.New("policy").Parse(resourceTemplate)
			if err != nil {
				return nil, err
			}

			friendlyResourceName, err := parseResourceFromTemplate(tmpl, vars)
			if err != nil {
				friendlyResourceName = "unknown"
			}

			// determine whether we have an infrastructure-as-code definition for the resource
			// TODO: confirm whether this lookup needs to be modified for more complex resources
			// e.g. DynamoDB tables or KMS keys.
			// cdkResource := a.auditor.GetCDKResourceByPhysicalID(friendlyResourceName)

			var resBytes bytes.Buffer
			err = tmpl.Execute(&resBytes, vars)
			if err != nil {
				return nil, err
			}

			renderedResources = append(renderedResources, resBytes.String())

			resources = append(resources, CloudResourceInstance{
				ID:   uuid.NewString(),
				Name: friendlyResourceName,
				// CDKResource: cdkResource,
				ARN: resBytes.String(),
			})
		}

		iamStatement := policies.AWSIAMStatement{
			Sid:      "iamzero" + strings.Replace(uuid.NewString(), "-", "", -1),
			Effect:   "Allow",
			Action:   statement.Action,
			Resource: renderedResources,
		}
		iamStatements = append(iamStatements, iamStatement)
	}

	id := uuid.NewString()

	// build a recommended AWS policy
	policy := policies.AWSIAMPolicy{
		Version:   "2012-10-17",
		Id:        &id,
		Statement: iamStatements,
	}

	roleName, err := GetRoleOrUserNameFromARN(e.Identity.Role)
	if err != nil {
		return nil, err
	}

	advice := LeastPrivilegePolicy{
		AWSPolicy: policy,
		Comment:   r.Comment,
		ID:        id,
		RoleName:  roleName,
		Resources: resources,
	}
	return &advice, nil
}

func (a *LeastPrivilegePolicy) GetID() string {
	return a.ID
}

func (a *LeastPrivilegePolicy) getDescription() []Description {
	return []Description{
		{
			AppliedTo: a.RoleName,
			Type:      "IAM Policy",
			Policy:    a.AWSPolicy,
		},
	}
}

func (a *LeastPrivilegePolicy) Details() RecommendationDetails {
	desc := a.getDescription()
	details := RecommendationDetails{
		ID:          a.ID,
		Comment:     a.Comment,
		Resources:   a.Resources,
		Description: desc,
	}
	return details
}

// parseResourceFromTemplate parses the resource out of a templated advisory.
// For example, if "arn:aws:s3:::{{ .Bucket }}/{{ .Key }}" is the template
// and the provided variables are Bucket=test-bucket and Key=test-key,
// the returned resource string should be test-bucket/test-key
//
// This method ignores the Account and Region fields to provide a human-friendly for the parsed resource
func parseResourceFromTemplate(t *template.Template, vars map[string]interface{}) (string, error) {
	resourceVals := []string{}

	// ignore the error here, we just try and match any variables used with the provided ones
	varsUsed, _ := requiredTemplateVars(t)
	for _, v := range varsUsed {
		if v == "Account" || v == "Region" {
			// don't include the Account or Region variables so that our
			// resource name is human-friendly
			continue
		}
		str, ok := vars[v].(string)
		if ok {
			resourceVals = append(resourceVals, str)
		}
	}
	if len(resourceVals) == 0 {
		return "", errors.New("resource could not be parsed")
	}
	resource := strings.Join(resourceVals, "/")
	return resource, nil
}

// Extract the template vars required from *simple* templates.
// Only works for top level, plain variables. Returns all problematic parse.Node as errors.
// Reference: https://stackoverflow.com/a/62224127
func requiredTemplateVars(t *template.Template) ([]string, []error) {
	var res []string
	var errors []error
	ln := t.Tree.Root
Node:
	for _, n := range ln.Nodes {
		if nn, ok := n.(*parse.ActionNode); ok {
			p := nn.Pipe
			if len(p.Decl) > 0 {
				errors = append(errors, fmt.Errorf("Node %v not supported", n))
				continue Node
			}
			for _, c := range p.Cmds {
				if len(c.Args) != 1 {
					errors = append(errors, fmt.Errorf("Node %v not supported", n))
					continue Node
				}
				if a, ok := c.Args[0].(*parse.FieldNode); ok {
					if len(a.Ident) != 1 {
						errors = append(errors, fmt.Errorf("Node %v not supported", n))
						continue Node
					}
					res = append(res, a.Ident[0])
				} else {
					errors = append(errors, fmt.Errorf("Node %v not supported", n))
					continue Node
				}

			}
		} else {
			if _, ok := n.(*parse.TextNode); !ok {
				errors = append(errors, fmt.Errorf("Node %v not supported", n))
				continue Node
			}
		}
	}
	return res, errors
}
