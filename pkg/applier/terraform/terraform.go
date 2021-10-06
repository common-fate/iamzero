package applier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

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
	Blocks              *Blocks
	StateFileResources  *StateFileResources
	// Define a role to assume when attempting to fetch remote state from s3, alternatively
	// don't set this attribute and the default aws credentials will be used if available
	AssumeRole *string
}
type Block struct {
	ParentModuleBlock *Block
	File              *hclwrite.File
	RawBlock          *hclwrite.Block
	Path              string
	AddressInFile     string
}
type Blocks map[string]*Block

type StateFileResourceRef struct {
	Key               string
	StateFileResource *StateFileResource
}
type StateFileResources map[string]StateFileResourceRef

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

	b := make(Blocks)
	if err = b.Init(t.getRootFilePath(), t.FileHandler); err != nil {
		return errors.Wrap(err, "initialising applier, initialising blocks failed")
	}
	t.Blocks = &b

	t.StateFileResources = stateFile.ParseStateFileToStateFileResources()

	return nil
}

// this will create a mapping of ARN as keys to module paths that can be used to index the Blocks map
func (stateFile *StateFile) ParseStateFileToStateFileResources() *StateFileResources {
	s := make(StateFileResources)
	for _, resource := range stateFile.Resources {
		for _, instance := range resource.Instances {
			key := strings.Join(append([]string{resource.Type}, resource.Name), ".")
			if resource.Module != nil {
				key = strings.Join(append([]string{*resource.Module, resource.Type}, resource.Name), ".")
			}

			s[instance.Attributes.Arn] = StateFileResourceRef{Key: key, StateFileResource: &resource}
		}
	}
	return &s
}

func (b *Blocks) Init(rootFilePath string, fileHandler *FileHandler) error {
	return TerraformDFS(b, rootFilePath, "", nil, fileHandler)
}

func TerraformDFS(blocks *Blocks, filePath string, modulePath string, parentModuleBlock *Block, fileHandler *FileHandler) error {
	if blocks == nil {
		return fmt.Errorf("blocks input cannot be nil")
	}

	hclfile, err := fileHandler.OpenFile(filePath, false)
	if err != nil {
		return err
	}
	for _, block := range hclfile.Body().Blocks() {
		bl := Block{Path: filePath, File: hclfile, RawBlock: block, ParentModuleBlock: parentModuleBlock, AddressInFile: strings.Join(block.Labels(), ".")}
		// The key will be in this format
		// "module.ec2.{TYPE}.{INSTANCE_ID}"
		key := strings.Join(append([]string{modulePath}, block.Labels()...), ".")
		if modulePath == "" {
			key = strings.Join(block.Labels(), ".")
		}

		if block.Type() == "module" {
			key = strings.Join(append([]string{modulePath, block.Type()}, block.Labels()...), ".")
			if modulePath == "" {
				key = strings.Join(append([]string{block.Type()}, block.Labels()...), ".")
			}
			if ok, dir := ModuleBlockHasLocalSource(block); ok {
				// dfs
				// the filepath param joins the current filepath with the source definition so that the filepath is a path from the working directory of the project
				if err = TerraformDFS(blocks, path.Join(path.Dir(filePath), *dir, "main.tf"), key, &bl, fileHandler); err != nil {
					return err
				}
			}
		}

		(*blocks)[key] = &bl
	}
	return nil
}

// returns boolean if the module has a local source, and returns the cleaned directory path
func ModuleBlockHasLocalSource(block *hclwrite.Block) (bool, *string) {
	if block.Type() != "module" {
		return false, nil
	}
	moduleSourceAttr := block.Body().GetAttribute("source")
	if moduleSourceAttr == nil {
		return false, nil
	}

	moduleSource := CleanAttributeExpression(moduleSourceAttr)

	// https://www.terraform.io/docs/language/modules/sources.html#local-paths
	if strings.HasPrefix(moduleSource, "./") || strings.HasPrefix(moduleSource, "../") {
		return true, &moduleSource
	}
	return false, nil
}

// returns the expression of the attribute stripped of leading and trailing whitespace and " doublequotes
// in terraform , only double quotes are allowed to represent a string litteral
func CleanAttributeExpression(attr *hclwrite.Attribute) string {
	return strings.Trim(string(attr.Expr().BuildTokens(nil).Bytes()), ` "`)
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
func (t *TerraformIAMPolicyApplier) IsBlockInRoot(b *Block) bool {
	return t.getRootFilePath() == b.Path
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
func (fh FileHandler) OpenFile(filepath string, createIfNotExist bool) (*hclwrite.File, error) {
	cleanPath := path.Clean(filepath)
	if val, ok := fh.HclFiles[cleanPath]; ok {
		return val, nil
	}
	var err error
	_, fh.HclFiles[cleanPath], err = OpenAndParseHclFile(cleanPath, createIfNotExist)
	if err != nil {
		return nil, err
	}
	return fh.HclFiles[cleanPath], nil

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
func (t *TerraformIAMPolicyApplier) FindAndRemovePolicyAttachmentsForRole(stateFileRole StateFileResourceRef) error {
	if len(stateFileRole.StateFileResource.Instances) != 1 {
		return errors.New("failed to find a role name to search for when trying to remove policy attachemnts for role, check the provided statefile resource is correct")
	}
	attachments := t.FindPolicyAttachmentsInStateFileByRoleName(stateFileRole.StateFileResource.Instances[0].Attributes.Id)
	for _, attachment := range attachments {
		policyAttachmentBlock := t.Blocks.GetBlock(attachment.Key)
		RemovePolicyAttachmentRole(policyAttachmentBlock.RawBlock, *stateFileRole.StateFileResource)

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

func (b *Blocks) GetBlock(modulePath string) *Block {
	return (*b)[modulePath]
}
func (s *StateFileResources) Get(arn string) StateFileResourceRef {
	return (*s)[arn]
}
func (t *TerraformIAMPolicyApplier) PlanTerraformFinding() (*applier.PendingChanges, error) {
	// @TODO mabye shift this to a seperate function
	if t.Blocks == nil || t.StateFileResources == nil {
		return nil, fmt.Errorf("applier not initialiser")
	}

	// @TODO handle failures here
	awsIamBlock := t.Blocks.GetBlock(t.StateFileResources.Get(t.Finding.Role).Key)
	if awsIamBlock != nil {
		// Remove any managed policy attachments for this role
		err := t.FindAndRemovePolicyAttachmentsForRole(t.StateFileResources.Get(t.Finding.Role))
		if err != nil {
			return nil, err
		}

		// Apply the finding by appending inline policies to the role
		err = t.ApplyFindingToBlock(awsIamBlock)
		if err != nil {
			return nil, err
		}
		return t.PendingChanges(), nil
	}
	// If we don't find the matching role, either something went wrong in our code, or the statefile doesn't match the source code.
	// the user probably needs to run `terraform plan` again
	return nil, fmt.Errorf("an error occurred finding the matching iam role in your terraform project, your state file may be outdated, try running 'terraform plan'")

}
func (t *TerraformIAMPolicyApplier) ApplyFindingToBlock(awsIamBlock *Block) error {
	newInlinePolicies := []*hclwrite.Block{}
	existingInlinePoliciesToRemove := []*hclwrite.Block{}
	for _, nestedBlock := range awsIamBlock.RawBlock.Body().Blocks() {
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
			awsResource := t.Blocks.GetBlock(t.StateFileResources.Get(splitArn[0]).Key)
			if awsResource != nil {
				/*
					THE BELOW SCENARIOS ONLY SUPPORT A FLAT PROJECT STRUCTURE WHERE THERE IS ONLY 1 LEVEL OF MODULE ABSTRACTION

					MAIN.TF
						->MODULES
							->EC2
								MAIN.TF

				*/

				// if both are in the same file, needs to be nil safe
				if awsResource.Path == awsIamBlock.Path {
					// both in same file
					// standard method

					arn = awsResource.AddressInFile + ".arn"
					// if there is a specific resource then join it to the resource arn
					if len(splitArn) > 1 && splitArn[1] != "*" {
						// This adds a join statement to the terraform so that we refer to the correct bucket arn but add the specific resource correctly if it was specified in the finding
						// https://www.terraform.io/docs/language/functions/join.html
						arn = fmt.Sprintf(`join("/", [%s,"%s"])`, arn, strings.Join(splitArn[1:], "/"))
					}

				} else if !t.IsBlockInRoot(awsResource) && !t.IsBlockInRoot(awsIamBlock) {
					// resources are in different files
					// create and output for the resource
					// create a variable for the role module
					//do the plumbing
					outputsFilePath := filepath.Join(filepath.Dir(awsResource.Path), "outputs.tf")
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}

					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.AddressInFile, "arn"), "IAMZero generated output for resource", awsResource.AddressInFile+".arn")

					moduleDefinitionInRoot := awsResource.ParentModuleBlock
					resourcePathInRootModule := ""
					resourcePathInRootModule = strings.Join([]string{"module", moduleDefinitionInRoot.AddressInFile, outputName}, ".")

					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(awsIamBlock.Path), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}

					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.AddressInFile, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot = awsIamBlock.ParentModuleBlock

					AppendTraversalAttributeToBlock(moduleDefinitionInRoot.RawBlock, variableName, resourcePathInRootModule)
					arn = "var." + variableName

				} else if !t.IsBlockInRoot(awsResource) && t.IsBlockInRoot(awsIamBlock) {

					// role is in root,  resource is in another file
					// create an output from the resource definition
					// refer to it in the root inline policy

					outputsFilePath := filepath.Join(filepath.Dir(awsResource.Path), "outputs.tf")
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}
					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.AddressInFile, "arn"), "IAMZero generated output for resource", awsResource.AddressInFile+".arn")

					moduleDefinitionInRoot := awsResource.ParentModuleBlock
					if moduleDefinitionInRoot != nil {

						arn = strings.Join([]string{"module", moduleDefinitionInRoot.AddressInFile, outputName}, ".")
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsResource.Path)
					}

				} else if t.IsBlockInRoot(awsResource) && !t.IsBlockInRoot(awsIamBlock) {
					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(awsIamBlock.Path), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}
					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.AddressInFile, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot := awsIamBlock.ParentModuleBlock
					if moduleDefinitionInRoot != nil {
						AppendTraversalAttributeToBlock(moduleDefinitionInRoot.RawBlock, variableName, awsResource.AddressInFile+".arn")
						arn = "var." + variableName
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsIamBlock.Path)
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
		awsIamBlock.RawBlock.Body().RemoveBlock(blockToRemove)
	}
	// append the new blocks(inline policies) to this role
	for _, blockToAdd := range newInlinePolicies {
		awsIamBlock.RawBlock.Body().AppendBlock(blockToAdd)
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

func (tr TerraformResource) SplitArn() []string {
	return strings.Split(*tr.ARN, "/")
}

func sliceContains(slice []string, compareTo string) bool {
	for _, elem := range slice {
		if elem == compareTo {
			return true
		}
	}
	return false
}
func (t *TerraformIAMPolicyApplier) FindPolicyAttachmentsInStateFileByRoleName(name string) []*StateFileResourceRef {
	resourceType := "aws_iam_policy_attachment"
	matches := make(map[string]*StateFileResourceRef)

	// First checks if its in the root modules
	for _, r := range *t.StateFileResources {
		if r.StateFileResource.Type == resourceType && sliceContains(r.StateFileResource.Instances[0].Attributes.Roles, name) {
			matches[r.Key] = &r
		}
	}

	values := make([]*StateFileResourceRef, 0, len(matches))
	for _, element := range matches {
		values = append(values, element)
	}

	return values
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
		stateFileBytes, err := FetchStateFileFromS3BackendBlockDefinition(context.Background(), block, t.AssumeRole)
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

// Accepts a role arn as an input, nil pointer will use the default account
func FetchStateFileFromS3BackendBlockDefinition(ctx context.Context, block *hclwrite.Block, role *string) ([]byte, error) {
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
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return []byte{}, err
	}
	stsSvc := sts.NewFromConfig(cfg)

	// If a role is provided this will attempt to assume the role
	if role != nil {
		creds := stscreds.NewAssumeRoleProvider(stsSvc, *role)
		cfg.Credentials = aws.NewCredentialsCache(creds)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	svc := s3.NewFromConfig(cfg)
	result, err := svc.GetObject(ctx, input)
	if err != nil {
		return []byte{}, errors.Wrap(err, "failed to fetch remote state from s3")

	}
	defer result.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, result.Body); err != nil {
		return []byte{}, errors.Wrap(err, "failed reading fetched state file contents")
	}
	return buf.Bytes(), nil
}

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

func IsBlockInlinePolicy(block *hclwrite.Block) bool {
	return block.Type() == "inline_policy"
}
