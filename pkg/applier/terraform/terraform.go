package applier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/common-fate/iamzero/pkg/applier"
	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

// TerraformFinding is proposed Terraform source code changes recommended by IAM Zero
type TerraformFinding struct {
	FindingID       string                    `json:"findingId"`
	Role            string                    `json:"role"`
	Recommendations []TerraformRecommendation `json:"recommendations"`
}

type TerraformRecommendation struct {
	Type       string               `json:"type"`
	Statements []TerraformStatement `json:"statements"`
}

type TerraformStatement struct {
	Resources []TerraformResource `json:"resources"`
	Actions   []string            `json:"actions"`
}

type TerraformResource struct {
	Reference string  `json:"reference"`
	ARN       *string `json:"arn,omitempty"`
}

type StateFile struct {
	Resources []StateFileResource `json:"resources"`
}
type StateFileResource struct {
	Type string `json:"type"`
	Name string `json:"name"`
	// The module property will be nil if this resource is in the root file
	Module    *string                     `json:"module"`
	Instances []StateFileResourceInstance `json:"instances"`
}
type StateFileResourceInstance struct {
	Attributes StateFileAttribute `json:"attributes"`
}
type StateFileAttribute struct {
	Arn string `json:"arn"`
	//This is the name
	Id string `json:"id"`
	// Will only be populated for some resource types
	Roles []string `json:"roles"`
}

type StateFileResourceBlock struct {
	StateFileResource
	FilePath         string
	FilePathFromRoot string
}

// FileHandler is used to manage opening and parsing HCL files for use during planning and applying
//
// This helper simplifies the process of making many changes to the same files and applying them all in a single step
type FileHandler struct {
	HclFiles  map[string]*hclwrite.File
	StateFile *StateFile
}

type AwsIamBlock struct {
	*hclwrite.Block
}

// TerraformIAMPolicyApplier implements the PolicyApplier interface
//
// an Applier instance is intended to operate on a single terraform project path
//
// To operate on a different project, create a new instance with a different project path
type TerraformIAMPolicyApplier struct {
	AWSIAMPolicyApplier applier.AWSIAMPolicyApplier
	Finding             *TerraformFinding
	FileHandler         *FileHandler
	StateFile           *StateFile
}

var MAIN_TERRAFORM_FILE = "main.tf"
var LOCAL_TERRAFORM_STATE_FILE = "terraform.tfstate"

// Returns a formatted name for the type of project this applier is for
//
// Used by the applier CLI to compose logging output
func (t *TerraformIAMPolicyApplier) GetProjectName() string { return "Terraform" }

// Initializes the FileHandler
//
// Attempts to read and parse the Terraform state for the project in the current working directory as specified by the
// TerraformIAMPolicyApplier.AWSIAMPolicyApplier.ProjectPath
//
// Will return any errors encountered while loading the statefile
func (t *TerraformIAMPolicyApplier) Init() error {
	// Init File handler to manage reading and writing
	t.FileHandler = &FileHandler{HclFiles: make(map[string]*hclwrite.File)}

	// load the statefile if found at TerraformIAMPolicyApplier.AWSIAMPolicyApplier.ProjectPath
	stateFile, err := t.ParseTerraformState()
	if err != nil {
		return err
	}
	t.StateFile = stateFile
	return nil
}

// tests wether the TerraformIAMPolicyApplier.AWSIAMPolicyApplier.ProjectPath contains a main.tf file
func (t *TerraformIAMPolicyApplier) Detect() bool {
	_, err := os.Stat(t.getRootFilePath())
	return err == nil
}

// Processes policy and actions into a format that is simple for the applier to use
// the result is stored internally
func (t *TerraformIAMPolicyApplier) CalculateFinding(policy *recommendations.Finding, actions []recommendations.AWSAction) {
	t.calculateTerraformFinding(policy, actions)
}

// This will return the results of applying the stored finding.
//
// The CalculateFinding method must be run before calling this method
func (t *TerraformIAMPolicyApplier) Plan() (*applier.PendingChanges, error) {
	return t.PlanTerraformFinding()
}

// This will write the pending changes to disk
func (t *TerraformIAMPolicyApplier) Apply(changes *applier.PendingChanges) error {
	// Writes the changes to the files
	for _, change := range *changes {
		err := ioutil.WriteFile(change.Path, []byte(change.Contents), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// Returns the full path to the root terraform main file
func (t *TerraformIAMPolicyApplier) getRootFilePath() string {
	return path.Join(t.AWSIAMPolicyApplier.ProjectPath, MAIN_TERRAFORM_FILE)
}

// Creates a single finding from enabled alerts
func (t *TerraformIAMPolicyApplier) calculateTerraformFinding(policy *recommendations.Finding, actions []recommendations.AWSAction) {

	terraformFinding := TerraformFinding{
		FindingID: policy.ID,
		Role:      policy.Identity.Role,

		Recommendations: []TerraformRecommendation{
			// {
			// 	Type: "IAMInlinePolicy",
			// 	Statements: []TerraformStatement{
			// 		{
			// 			Resources: []TerraformResource{
			// 				{
			// 					Reference: bucketArn,
			// 					Type:      "AWS::S3::Bucket", ARN: &bucketArn,
			// 				},
			// 			},
			// 			Actions: actionsDemo,
			// 		},
			// 	},
			// },
		},
	}

	// I copied this and modified it from the CDK example, it is subject to the same TODO comments as CDK above
	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			rec := TerraformRecommendation{
				Type:       "IAMInlinePolicy",
				Statements: []TerraformStatement{},
			}
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				if ok {
					for _, s := range policy.Statement {
						terraformStatement := TerraformStatement{
							Actions: s.Action,
						}
						for _, resource := range alert.Resources {

							var terraformResource TerraformResource
							if resource.CDKResource == nil {

								terraformResource = TerraformResource{
									Reference: "IAM",
									ARN:       &resource.ARN,
								}
							}
							terraformStatement.Resources = append(terraformStatement.Resources, terraformResource)
						}
						rec.Statements = append(rec.Statements, terraformStatement)
					}
				}
			}
			terraformFinding.Recommendations = append(terraformFinding.Recommendations, rec)
		}
	}
	t.Finding = &terraformFinding
}

// Returns true if this StateFileResource is in the root directory by checking wether the Module property is nil
func (sfr StateFileResource) IsInRoot() bool {
	return sfr.Module == nil
}

// Returns an iamzero formatted variable name
//
// iamzero-variable_<resourceName>_<propertyType>
func GenerateVariableName(resourceName string, propertyType string) string {
	return strings.Join([]string{"iamzero-variable", resourceName, propertyType}, "_")
}

// Returns an iamzero formatted output name
//
// iamzero-output_<resourceName>_<propertyType>
func GenerateOutputName(resourceName string, propertyType string) string {
	return strings.Join([]string{"iamzero-output", resourceName, propertyType}, "_")
}

// Returns an hcl.Traversal which can be written to an hcl file without including quotes
// This is used when setting an attribute value to a path to a module or variable etc
//
//  address should be a "." seperated string
//
// e.g "modules.ec2.my_server.arn"
//
//  can be applied to the attribute 'name' using block.Body().SetAttributeTraversal
//
//
// resulting in 'name = modules.ec2.my_server.arn'
func TraversalFromAddress(address string) hcl.Traversal {
	splitAddress := strings.Split(address, ".")
	traversal := hcl.Traversal{
		hcl.TraverseRoot{Name: splitAddress[0]},
	}
	for _, val := range splitAddress[1:] {
		traversal = append(traversal, hcl.TraverseAttr{Name: val})
	}
	return traversal
}

// Appends an attribute to the provided Block
// the result will be something like 'name = modules.ec2.my_server.arn'
//
// resourcePath should be a "." seperated string e.g "modules.ec2.my_server.arn"
func AppendTraversalAttributeToBlock(block *hclwrite.Block, variableName string, resourcePath string) {
	block.Body().SetAttributeTraversal(variableName, TraversalFromAddress(resourcePath))
}

// This is intended to be used when operating on a variables.tf file
//
// This function firsts inspects the file to find a variable that already exists with the same name
// if it already exists, no changes are made, if it doesn exist it is appended to the body
func AppendVariableBlockIfNotExist(body *hclwrite.Body, name string, description string) string {
	block := FindBlockWithMatchingLabel(body.Blocks(), name)
	if block != nil {
		return name
	}
	AppendVariableBlock(body, name, description)
	return name

}

// Appends a variable block of type string with name and description
//
// variable "name" {
//  	type        = string
//  	description = "description"
// }
func AppendVariableBlock(body *hclwrite.Body, name string, description string) {
	body.AppendNewline()
	newBlock := hclwrite.NewBlock("variable", []string{name})
	newBlock.Body().SetAttributeValue("type", cty.StringVal("string"))
	newBlock.Body().SetAttributeValue("description", cty.StringVal(description))
	body.AppendBlock(newBlock)
	body.AppendNewline()
}

// This is intended to be used when operating on a outputs.tf file
//
// this function firsts inspects the file to find an output that already exists with the same value
//
// if it already exists, no changes are made, and the name of the matched output is returned
//
// if it doesn exist it is appended to the body and the new name is returned
func AppendOutputBlockIfNotExist(body *hclwrite.Body, name string, description string, value string) string {
	block := FindBlockWithMatchingValueAttribute(body.Blocks(), value)
	if block != nil {
		return block.Labels()[0]
	}
	AppendOutputBlock(body, name, description, value)
	return name
}

// Appends an output block of with name , value and description
//
// output "name" {
//  	value        = value
//  	description = "description"
// }
func AppendOutputBlock(body *hclwrite.Body, name string, description string, value string) {
	body.AppendNewline()
	newBlock := hclwrite.NewBlock("output", []string{name})
	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(value)}
	toks := hclwrite.Tokens{&t}
	newBlock.Body().SetAttributeRaw("value", toks)
	newBlock.Body().SetAttributeValue("description", cty.StringVal(description))
	body.AppendBlock(newBlock)
	body.AppendNewline()
}

// Open file will open or create an empty hclfile
// nothing is written to the filesystem by this method
func (fh FileHandler) OpenFile(path string, createIfNotExist bool) (*hclwrite.File, error) {
	if val, ok := fh.HclFiles[path]; ok {
		return val, nil
	}
	var err error
	_, fh.HclFiles[path], err = OpenAndParseHclFile(path, createIfNotExist)
	if err != nil {
		return nil, err
	}
	return fh.HclFiles[path], nil

}

// opens the file at the path and attempts to marshal it into a StateFile
// if there are no errors it writes the statefile back to the fh
func (fh FileHandler) OpenStateFile(path string) (*StateFile, error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s because %s", path, err)
	}
	fh.StateFile, err = MarshalStateFileToGo(src)
	if err != nil {
		return nil, err
	}
	return fh.StateFile, nil
}

// This method will scan the statefile for any instances of teh provided role in a policy attachment resource
// The policy attachment resource is then updated with the target role removed from the atachement policy
func (t *TerraformIAMPolicyApplier) FindAndRemovePolicyAttachmentsForRole(stateFileRole StateFileResource) error {
	if len(stateFileRole.Instances) != 1 {
		return errors.New("failed to find a role name to search for when trying to remove policy attachemnts for role, check the provided statefile resource is correct")
	}
	attachments := t.FindPolicyAttachmentsInStateFileByRoleName(stateFileRole.Instances[0].Attributes.Id)
	for key, element := range attachments {
		// for each terraform file, update all the modules
		hclFile, _ := t.FileHandler.OpenFile(key, false)
		for _, stateFileResource := range element {
			policyAttachmentBlock := FindBlockByModuleAddress(hclFile.Body().Blocks(), stateFileResource.Type+"."+stateFileResource.Name)
			RemovePolicyAttachmentRole(policyAttachmentBlock, stateFileRole)
		}
	}
	return nil
}

// return the formatted bytes for each open file
func (t *TerraformIAMPolicyApplier) PendingChanges() *applier.PendingChanges {

	pc := applier.PendingChanges{}
	for key, val := range t.FileHandler.HclFiles {
		pc = append(pc, applier.PendingChange{Path: key, Contents: string(hclwrite.Format(val.Bytes()))})
	}
	return &pc
}

func (t *TerraformIAMPolicyApplier) PlanTerraformFinding() (*applier.PendingChanges, error) {
	iamRoleStateFileResource, err := t.FindResourceInStateFileByArn(t.Finding.Role)
	if err != nil {
		return nil, err
	}
	hclFile, err := t.FileHandler.OpenFile(iamRoleStateFileResource.FilePath, false)
	if err != nil {
		return nil, err
	}
	awsIamBlock := FindIamRoleBlockByModuleAddress(hclFile.Body().Blocks(), iamRoleStateFileResource.Type+"."+iamRoleStateFileResource.Name)
	if awsIamBlock != nil {
		// Remove any managed policy attachments for this role
		err := t.FindAndRemovePolicyAttachmentsForRole(iamRoleStateFileResource.StateFileResource)
		if err != nil {
			return nil, err
		}
		rootHclFile, err := t.FileHandler.OpenFile(t.getRootFilePath(), false)
		if err != nil {
			return nil, err
		}
		// Apply the finding by appending inline policies to the role
		err = t.ApplyFindingToBlock(awsIamBlock, iamRoleStateFileResource, rootHclFile)
		if err != nil {
			return nil, err
		}
		return t.PendingChanges(), nil
	}
	// If we don't find the matching role, either something went wrong in our code, or the statefile doesn't match the source code.
	// the user probably needs to run `terraform plan` again
	return nil, fmt.Errorf("an error occurred finding the matching iam role in your terraform project, your state file may be outdated, try running 'terraform plan'")

}
func (t *TerraformIAMPolicyApplier) ApplyFindingToBlock(awsIamBlock *AwsIamBlock, iamRoleStateFileResource *StateFileResourceBlock, rootHclFile *hclwrite.File) error {
	newInlinePolicies := []*hclwrite.Block{}
	existingInlinePoliciesToRemove := []*hclwrite.Block{}
	for _, nestedBlock := range awsIamBlock.Body().Blocks() {
		if IsBlockInlinePolicy(nestedBlock) {
			existingInlinePoliciesToRemove = append(existingInlinePoliciesToRemove, nestedBlock)
		}
	}

	for _, reccomendation := range t.Finding.Recommendations {
		for i, statement := range reccomendation.Statements {
			newBlock := hclwrite.NewBlock("inline_policy", nil)
			actionsJson, _ := json.Marshal(statement.Actions)
			arn := ""

			// The ARN from the recommendation can contain "/*" on the end for an s3 bucket, to look this up in
			// @TODO probably need to verify whether we could get specific things here(as in specific objects in a bucket)

			splitArn := statement.Resources[0].SplitArn()
			awsResource, err := t.FindResourceInStateFileByArn(splitArn[0])
			if err == nil {
				/*
					THE BELOW SCENARIOS ONLY SUPPORT A FLAT PROJECT STRUCTURE WHERE THERE IS ONLY 1 LEVEL OF MODULE ABSTRACTION

					MAIN.TF
						->MODULES
							->EC2
								MAIN.TF

				*/

				// if both are in the same file, needs to be nil safe
				if (awsResource.IsInRoot() && iamRoleStateFileResource.IsInRoot()) || (awsResource.Module != nil && iamRoleStateFileResource.Module != nil && *awsResource.Module == *iamRoleStateFileResource.Module) {
					// both in same file
					// standard method
					arn = awsResource.GetAddressWithinFile() + ".arn"
					// if there is a specific resource then join it to the resource arn
					if len(splitArn) > 1 && splitArn[1] != "*" {
						// This adds a join statement to the terraform so that we refer to the correct bucket arn but add the specific resource correctly if it was specified in the finding
						// https://www.terraform.io/docs/language/functions/join.html
						arn = fmt.Sprintf(`join("/", [%s,"%s"])`, arn, strings.Join(splitArn[1:], "/"))
					}

				} else if !awsResource.IsInRoot() && !iamRoleStateFileResource.IsInRoot() {
					// resources are in different files
					// create and output for the resource
					// create a variable for the role module
					//do the plumbing
					outputsFilePath := filepath.Join(filepath.Dir(awsResource.FilePath), "outputs.tf")
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}

					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", awsResource.GetAddressWithinFile()+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), awsResource.FilePathFromRoot)
					resourcePathInRootModule := ""
					if moduleDefinitionInRoot != nil {

						resourcePathInRootModule = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsResource.FilePathFromRoot)
					}

					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}

					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot = FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), iamRoleStateFileResource.FilePathFromRoot)

					if moduleDefinitionInRoot != nil {
						AppendTraversalAttributeToBlock(moduleDefinitionInRoot, variableName, resourcePathInRootModule)
						arn = "var." + variableName
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", iamRoleStateFileResource.FilePathFromRoot)
					}

				} else if !awsResource.IsInRoot() && iamRoleStateFileResource.IsInRoot() {

					// role is in root,  resource is in another file
					// create an output from the resource definition
					// refer to it in the root inline policy

					outputsFilePath := filepath.Join(filepath.Dir(awsResource.FilePath), "outputs.tf")
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}
					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", awsResource.GetAddressWithinFile()+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), awsResource.FilePathFromRoot)
					if moduleDefinitionInRoot != nil {

						arn = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsResource.FilePathFromRoot)
					}

				} else if awsResource.IsInRoot() && !iamRoleStateFileResource.IsInRoot() {
					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}
					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), iamRoleStateFileResource.FilePathFromRoot)
					if moduleDefinitionInRoot != nil {
						AppendTraversalAttributeToBlock(moduleDefinitionInRoot, variableName, awsResource.Name+".arn")
						arn = "var." + variableName
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", iamRoleStateFileResource.FilePathFromRoot)
					}
				} else {
					// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
					arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
					t.AWSIAMPolicyApplier.Logger.Warnf("Resource with ARN(%s) is declared more than 1 level into the project tree, this is not yet supported. Using ARN directly\n", arn, t.StateFile)
				}

			} else {
				// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
				arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
				t.AWSIAMPolicyApplier.Logger.Warnf("Failed to find matching resource in state file for ARN(%s) in (%s) using ARN reference directly\n", arn, t.StateFile)
			}

			setInlinePolicyIamPolicy(newBlock, string(actionsJson), arn, "iamzero-generated-iam-policy-"+fmt.Sprint(i))
			newInlinePolicies = append(newInlinePolicies, newBlock)
		}
	}
	// Remove all existing inline policies
	for _, blockToRemove := range existingInlinePoliciesToRemove {
		awsIamBlock.Body().RemoveBlock(blockToRemove)
	}
	// append the new blocks(inline policies) to this role
	for _, blockToAdd := range newInlinePolicies {
		awsIamBlock.Body().AppendBlock(blockToAdd)
	}
	return nil

}

// Intended to be used on a policy attachment block
// this function will attempt to find a matching role eather by a string litteral or a traversal
// the role is removed and the attribute is updated
func RemovePolicyAttachmentRole(block *hclwrite.Block, stateFileRole StateFileResource) {
	line := string(block.Body().GetAttribute("roles").Expr().BuildTokens(hclwrite.Tokens{}).Bytes())
	line = strings.Trim(line, " []")
	roles := strings.Split(line, ",")
	filteredRoles := []string{}
	for _, role := range roles {
		// will either be a string litteral rolename or a reference to a role in tf
		trimmedRole := strings.Trim(role, " ")
		if !(strings.Contains(trimmedRole, stateFileRole.Type+"."+stateFileRole.Name) || (len(stateFileRole.Instances) == 1 && trimmedRole == `"`+stateFileRole.Instances[0].Attributes.Id+`"`)) {
			filteredRoles = append(filteredRoles, trimmedRole)
		}
	}

	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(fmt.Sprintf(`[%s]`, strings.Join(filteredRoles, ",")))}
	toks := hclwrite.Tokens{&t}
	block.Body().SetAttributeRaw("roles", toks)
}

//filters the blocks to find the target block by matching the address
// and address looks like "aws_iam_role.iamzero-overprivileged-role"
//
// for multiple files, this address will refer to the file by is path e.g "modules.ec2.aws_iam_role.role-name"
func FindIamRoleBlockByModuleAddress(blocks []*hclwrite.Block, moduleAddress string) *AwsIamBlock {

	for _, block := range blocks {
		if IsBlockAwsIamRole(block) {
			if resourceLabelToString(block.Labels()) == moduleAddress {
				return &AwsIamBlock{block}
			}
		}
	}
	return nil
}

// compares the "source" attribute of a module block in terraform, this accepts a full filepath or the directory path to the module file
func FindModuleBlockBySourcePath(blocks []*hclwrite.Block, moduleFolderPath string) *hclwrite.Block {
	for _, block := range blocks {
		if block.Type() == "module" {
			source := block.Body().GetAttribute("source")
			if source != nil && filepath.Dir(strings.Trim(string(block.Body().GetAttribute("source").Expr().BuildTokens(nil).Bytes()), ` "`)) == filepath.Dir(moduleFolderPath) {
				return block
			}
		}
	}
	return nil
}

// address should be a "." separated string "modules.ec2.example"
func FindBlockByModuleAddress(blocks []*hclwrite.Block, moduleAddress string) *hclwrite.Block {
	for _, block := range blocks {

		if resourceLabelToString(block.Labels()) == moduleAddress {
			return block
		}
	}
	return nil
}

func FindBlockWithMatchingValueAttribute(blocks []*hclwrite.Block, valueString string) *hclwrite.Block {
	for _, block := range blocks {
		value := block.Body().GetAttribute("value")
		if value != nil && strings.Trim(string(value.Expr().BuildTokens(nil).Bytes()), " ") == valueString {
			return block
		}
	}
	return nil
}
func FindBlockWithMatchingLabel(blocks []*hclwrite.Block, name string) *hclwrite.Block {
	for _, block := range blocks {
		if len(block.Labels()) > 0 {
			value := block.Labels()[0]
			if value == name {
				return block
			}
		}
	}
	return nil
}

func resourceLabelToString(labels []string) string {
	return strings.Join(labels[:], ".")
}

func (tr TerraformResource) SplitArn() []string {
	return strings.Split(*tr.ARN, "/")
}

func (t *TerraformIAMPolicyApplier) FindResourceInStateFileByArn(arn string) (*StateFileResourceBlock, error) {
	return t.FindResourceInStateFileBase(arn, func(s StateFileResource) string {
		// @TODO make this safe for multi instance
		return s.Instances[0].Attributes.Arn
	})
}
func (t *TerraformIAMPolicyApplier) FindResourceInStateFileByName(name string) (*StateFileResourceBlock, error) {
	return t.FindResourceInStateFileBase(name, func(s StateFileResource) string {
		// @TODO make this safe for multi instance
		return s.Instances[0].Attributes.Id
	})
}
func sliceContains(slice []string, compareTo string) bool {
	for _, elem := range slice {
		if elem == compareTo {
			return true
		}
	}
	return false
}
func (t *TerraformIAMPolicyApplier) FindPolicyAttachmentsInStateFileByRoleName(name string) map[string][]StateFileResource {
	terraformFilePath := t.getRootFilePath()
	resourceType := "aws_iam_policy_attachment"
	matches := make(map[string][]StateFileResource)

	// First checks if its in the root modules
	for _, r := range t.StateFile.Resources {
		if r.Type == resourceType && sliceContains(r.Instances[0].Attributes.Roles, name) {
			if r.Module == nil {
				if matches[terraformFilePath] == nil {
					matches[terraformFilePath] = []StateFileResource{}
				}
				matches[terraformFilePath] = append(matches[terraformFilePath], r)
			} else {
				// This line takes the module attribute "module.ec2" and splits it to remove "module" and replace with "modules"
				// may need to revisit this when we look at supporting nested modules
				terraformFilePath = filepath.Join(t.AWSIAMPolicyApplier.ProjectPath, "modules", filepath.Join(strings.Split(*r.Module, ".")[1:]...), MAIN_TERRAFORM_FILE)
				if matches[terraformFilePath] == nil {
					matches[terraformFilePath] = []StateFileResource{}
				}
				matches[terraformFilePath] = append(matches[terraformFilePath], r)
			}

		}
	}

	return matches
}

func (sfr *StateFileResource) GetAddressWithinFile() string {
	return strings.Join([]string{sfr.Type, sfr.Name}, ".")

}
func (t *TerraformIAMPolicyApplier) FindResourceInStateFileBase(value string, attributefn func(s StateFileResource) string) (*StateFileResourceBlock, error) {
	/*
		I added this to allow matching of any of the properties of the resource by passing in a function to select the attribute
	*/
	terraformFilePath := t.getRootFilePath()
	for _, r := range t.StateFile.Resources {
		// its in the root
		if r.Module == nil && attributefn(r) == value {
			return &StateFileResourceBlock{r, terraformFilePath, "./"}, nil
		} else if r.Module != nil && attributefn(r) == value {
			// This line takes the module attribute "module.ec2" and splits it to remove "module" and replace with "modules"
			// may need to revisit this when we look at supporting nested modules
			terraformFilePathFromRoot := filepath.Join("modules", filepath.Join(strings.Split(*r.Module, ".")[1:]...), MAIN_TERRAFORM_FILE)
			terraformFilePath = filepath.Join(t.AWSIAMPolicyApplier.ProjectPath, terraformFilePathFromRoot)

			return &StateFileResourceBlock{r, terraformFilePath, terraformFilePathFromRoot}, nil
		}
	}

	return nil, fmt.Errorf("could not find a matching resource in the state file for %s", value)
}

// This method will determine the configuration used for storing state
// it will then open and parse the state
// currently handles the default s3 storage definition
// and local state
func (t *TerraformIAMPolicyApplier) ParseTerraformState() (*StateFile, error) {
	localState := false

	// Attempt to open local state file, if found, set the flag to true for use later
	_, err := os.Stat(path.Join(t.AWSIAMPolicyApplier.ProjectPath, LOCAL_TERRAFORM_STATE_FILE))
	if err == nil {
		localState = true
	}

	// open main.tf and interrogat the backend config if it exists
	// if no backend config exists, check if the terraform.tfstate file exists
	// else retrun error
	mainFile, err := t.FileHandler.OpenFile(t.getRootFilePath(), false)
	if err != nil {
		return nil, errors.Wrap(err, "error while trying to parse state")
	}
	terraform := mainFile.Body().FirstMatchingBlock("terraform", []string{})
	if terraform == nil {
		return nil, errors.New("could not find terraform block while trying to parse terraform state")
	}
	block := terraform.Body().FirstMatchingBlock("backend", []string{"s3"})
	if block == nil {
		if localState {
			return t.FileHandler.OpenStateFile(path.Join(t.AWSIAMPolicyApplier.ProjectPath, "terraform.tfstate"))
		}
	} else {
		// fetch state file from s3
		stateFileBytes, err := FetchStateFileFromS3BackendBlockDefinition(block)
		if err != nil {
			return nil, errors.Wrap(err, "error while trying to fetch state from s3 bucket")
		}
		stateFile, err := MarshalStateFileToGo(stateFileBytes)
		if err != nil {
			return nil, errors.Wrap(err, "error while trying to marshal state fetched from s3 bucket")
		}
		return stateFile, nil
	}

	return nil, errors.New("failed to read state")

}

func FetchStateFileFromS3BackendBlockDefinition(block *hclwrite.Block) ([]byte, error) {
	bucketAttr := block.Body().GetAttribute("bucket")
	keyAttr := block.Body().GetAttribute("key")
	regionAttr := block.Body().GetAttribute("region")

	if bucketAttr == nil || keyAttr == nil || regionAttr == nil {
		return []byte{}, errors.New("remote state properties missing, could not fetch file")
	}

	// Strip the " quotes from the expressions
	bucket := strings.Trim(string(bucketAttr.Expr().BuildTokens(nil).Bytes()), ` "`)
	key := strings.Trim(string(keyAttr.Expr().BuildTokens(nil).Bytes()), ` "`)
	// region := string(regionAttr.Expr().BuildTokens(nil).Bytes())
	// using this example from aws
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#example_S3_GetObject_shared00
	session, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create aws session while parsing state")
	}
	svc := s3.New(session)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				return []byte{}, errors.Wrap(aerr, s3.ErrCodeNoSuchKey)
			case s3.ErrCodeInvalidObjectState:
				return []byte{}, errors.Wrap(aerr, s3.ErrCodeInvalidObjectState)
			default:
				return []byte{}, aerr
			}
		} else {
			return []byte{}, errors.Wrap(err, "failed to fetch remote state from s3")
		}

	}
	defer result.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, result.Body); err != nil {
		return []byte{}, errors.Wrap(err, "failed reading fetched state file contents")
	}
	return buf.Bytes(), nil
}

// // This method will attempt to run `terraform init` and `terraform show` cli commands with the directory path specified by TerraformIAMPolicyApplier.AWSIAMPolicyApplier.ProjectPath
// // These function will correctly fetch the state from either local or external state management, assuming the correct credentials exist for remote state
// func (t *TerraformIAMPolicyApplier) parseTerraformState() (*StateFile, error) {

// 	// JSON output via the -json option requires Terraform v0.12 or later.
// 	// https://www.terraform.io/docs/cli/commands/show.html
// 	_, err := exec.Command("terraform", "-chdir=./"+t.AWSIAMPolicyApplier.ProjectPath, "init").Output()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to init terraform ' %s", err)
// 	}
// 	out, err := exec.Command("terraform", "-chdir=./"+t.AWSIAMPolicyApplier.ProjectPath, "show", "-json").Output()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to load stateFile using 'terraform show -json' %s", err)
// 	}
// 	return MarshalStateFileToGo(out)
// }

func MarshalStateFileToGo(stateFileBytes []byte) (*StateFile, error) {
	var stateFile StateFile
	err := json.Unmarshal(stateFileBytes, &stateFile)
	if err != nil {
		return nil, err
	}
	// Set default values for StateFileResourceInstance Module property
	// if teh module property is not present, that means that the address is the root

	return &stateFile, nil
}

func setInlinePolicyIamPolicy(block *hclwrite.Block, action string, resource string, name string) {
	// @TODO if hclwrite add a simple way to write function values like this we may switch over,
	// However for now it seems this is the simplest way to add a function block to HCL using the hclwite package
	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(fmt.Sprintf(`jsonencode({
        Version = "2012-10-17"
        Statement = [
          {
            Action   = %s
            Effect   = "Allow"
            Resource = %s
          },
        ]
      })`, action, resource))}
	toks := hclwrite.Tokens{&t}
	block.Body().SetAttributeRaw("policy", toks)
	block.Body().SetAttributeValue("name", cty.StringVal(name))
}

// if createIfNotExist is true then this function will return a new empty hclwrite file if it is not found at the path
//
// returns fileCreated(bool), file, error
func OpenAndParseHclFile(filePath string, createIfNotExist bool) (bool, *hclwrite.File, error) {

	// read file bytes
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) && createIfNotExist {
			return true, hclwrite.NewEmptyFile(), nil
		}
		return false, hclwrite.NewEmptyFile(), fmt.Errorf("failed to read file %s because %s", filePath, err)
	}
	hclfile, diagnostics := hclwrite.ParseConfig(src, filePath, hcl.InitialPos)
	if diagnostics != nil && diagnostics.HasErrors() {
		hclErrors := diagnostics.Errs()
		return false, hclwrite.NewEmptyFile(), fmt.Errorf("failed to parse hcl file %s because of errors %s", filePath, hclErrors)
	}
	return false, hclfile, nil
}

func ParseHclFileForAwsIamBlocks(hclfile *hclwrite.File) []*hclwrite.Block {
	var blocks []*hclwrite.Block
	for _, block := range hclfile.Body().Blocks() {
		if block.Type() == "resource" && IsBlockAwsIamRole(block) {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func IsBlockAwsIamRole(block *hclwrite.Block) bool {
	return len(block.Labels()) > 0 && block.Labels()[0] == "aws_iam_role"
}
func IsBlockInlinePolicy(block *hclwrite.Block) bool {
	return block.Type() == "inline_policy"
}
